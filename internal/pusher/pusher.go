package pusher

import (
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/expfmt"
)

type Client struct {
	URL   string
	Token string
}

// Push usando Gatherer (FORMA CORRECTA PARA PUSHGATEWAY)
func (c Client) PushGatherer(
	job string,
	instance string,
	labels map[string]string,
	g prometheus.Gatherer,
) error {
	// 1. Recolectar métricas
	mfs, err := g.Gather()
	if err != nil {
		return err
	}

	// 2. Formatear payload (texto)
	var out strings.Builder
	for _, mf := range mfs {
		if _, err := expfmt.MetricFamilyToText(&out, mf); err != nil {
			return err
		}
	}
	payload := out.String()

	// 3. Asegurar salto de línea final (Requerimiento estricto)
	if !strings.HasSuffix(payload, "\n") {
		payload += "\n"
	}

	// 4. Construir URL (job + instance + labels)
	// Path format: /metrics/job/<JOBNAME>/<LABEL_NAME>/<LABEL_VALUE>...
	u := strings.TrimRight(c.URL, "/") + "/metrics/job/" + url.QueryEscape(job)
	
	// Agregar instance como grouping key
	u += "/instance/" + url.QueryEscape(instance)

	// Agregar labels adicionales (Ordenados para consistencia)
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		u += "/" + url.QueryEscape(k) + "/" + url.QueryEscape(labels[k])
	}

	// 5. Crear Request HTTP
	req, err := http.NewRequest("PUT", u, strings.NewReader(payload))
	if err != nil {
		return err
	}

	// 6. Headers obligatorios
	req.Header.Set("Content-Type", "text/plain; version=0.0.4")
	req.Header.Set("Authorization", "Bearer "+c.Token)

	// 7. Enviar
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// 8. Validar respuesta
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("pushgateway returned status: %s", resp.Status)
	}

	return nil
}
