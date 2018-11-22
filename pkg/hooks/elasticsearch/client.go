package elasticsearch

import (
	"encoding/json"
	"fmt"
	"time"

	resty "gopkg.in/resty.v1"
)

var timeout = time.Duration(60 * time.Second)

type ESHealth struct {
	Status  string `json:"status"`
	Cluster string `json:"cluster"`
}

type ESSettings struct {
	Persistent ESPersistentSetting `json:"persistent"`
}

type ESPersistentSetting struct {
	Reallocation string `json:"cluster.routing.allocation.enable"`
}

type ESSettingsAck struct {
	Acknowledged bool `json:"acknowledged"`
}

type ESFlush struct {
	Shards ESFlushShards `json:"_shards"`
}

type ESFlushShards struct {
	Total      int64 `json:"total"`
	Successful int64 `json:"successful"`
	Failed     int64 `json:"failed"`
}

func flushSync(host string) error {
	_, err := resty.
		SetRedirectPolicy(resty.FlexibleRedirectPolicy(5)).
		SetRetryCount(3).
		SetTimeout(timeout).R().
		Post(fmt.Sprintf("http://%s:9200/_flush/synced", host))

	return err
}

func setAllocation(host string, target string) error {
	resp, err := resty.
		SetRedirectPolicy(resty.FlexibleRedirectPolicy(5)).
		SetRetryCount(3).
		SetTimeout(timeout).R().
		SetBody(ESSettings{
			Persistent: ESPersistentSetting{
				Reallocation: target,
			},
		}).
		Put(fmt.Sprintf("http://%s:9200/_cluster/settings", host))

	if err != nil {
		return err
	}

	if resp.StatusCode() != 200 {
		return fmt.Errorf("setAllocation http status code was %d for %s",
			resp.StatusCode(), host)
	}

	m := ESSettingsAck{}
	if err := json.Unmarshal(resp.Body(), &m); err != nil {
		return err
	}

	if !m.Acknowledged {
		return fmt.Errorf("setAllocation wasn't acknowledged for host %s", host)
	}

	return nil
}

func isGreen(host string) error {
	resp, err := resty.
		SetRedirectPolicy(resty.FlexibleRedirectPolicy(5)).
		SetRetryCount(3).
		SetTimeout(timeout).R().
		Get(fmt.Sprintf("http://%s:9200/_cat/health?format=json", host))

	if err != nil {
		return err
	}

	if resp.StatusCode() != 200 {
		return fmt.Errorf("/_cat/health http status code was %d for %s",
			resp.StatusCode(), host)
	}

	m := make([]ESHealth, 1)
	if err := json.Unmarshal(resp.Body(), &m); err != nil {
		return err
	}

	if m[0].Status != "green" {
		return fmt.Errorf("/_cat/health is %s for host %s", m[0].Status, host)
	}

	return nil
}
