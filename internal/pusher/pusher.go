package pusher

import (
	"fmt"
	"io"
	"log"
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
	// Usar PathEscape para segmentos de path
	u := strings.TrimRight(c.URL, "/") + "/metrics/job/" + url.PathEscape(job)

	// Agregar instance como grouping key
	u += "/instance/" + url.PathEscape(instance)

	// Agregar labels adicionales (Ordenados para consistencia)
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		u += "/" + url.PathEscape(k) + "/" + url.PathEscape(labels[k])
	}

	// 5. Crear Request HTTP
	req, err := http.NewRequest("PUT", u, strings.NewReader(payload))
	if err != nil {
		return err
	}

	// 6. Headers obligatorios
	req.Header.Set("Content-Type", "text/plain; version=0.0.4")
	req.Header.Set("Authorization", "Bearer "+c.Token)

	// Log para debug token (temporal/diagnóstico)
	authHeader := req.Header.Get("Authorization")
	maskedAuth := "Empty"
	if len(authHeader) > 15 {
		maskedAuth = authHeader[:15] + "..." + authHeader[len(authHeader)-5:]
	}
	log.Printf("Pushing to %s | Header: %s | TokenLen: %d", u, maskedAuth, len(c.Token))

	// Log para debug
	// log.Printf("Pushing metrics to %s", u)

	// 7. Enviar
	client := &http.Client{
		Transport: &http.Transport{
			ForceAttemptHTTP2: false,
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// 8. Validar respuesta
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("pushgateway error: status=%s body=%s", resp.Status, string(body))
	}

	return nil
}
