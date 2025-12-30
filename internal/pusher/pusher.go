package pusher

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
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

	headers := http.Header{}
	headers.Set("Authorization", "Bearer "+c.Token)

	p := push.New(c.URL, job).
		Client(&http.Client{}).
		Grouping("instance", instance).
		Header(headers).
		Gatherer(g)

	for k, v := range labels {
		p = p.Grouping(k, v)
	}

	return p.Push()
}
