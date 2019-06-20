package cloud_provider

import (
	"fmt"
	"testing"
	"time"

	"k8s.io/cloud-provider-baiducloud/pkg/cloud-sdk/bce"
	"k8s.io/cloud-provider-baiducloud/pkg/cloud-sdk/cce"
)

func TestInstance(t *testing.T) {
	cfg := &CloudConfig{
		AccessKeyID:     "8e2fdc833cf44b4895afd0bce14f43cf",
		SecretAccessKey: "7ae4ae1828694bbc814bb06fa87a43fa",
		Region:          "su",
		Endpoint:        "100.67.0.45:8693",
		ClusterID:       "c-PGBW8SAY",
	}
	bceConfig := bce.NewConfig(bce.NewCredentials(cfg.AccessKeyID, cfg.SecretAccessKey))
	bceConfig.Region = cfg.Region
	bceConfig.Timeout = 10 * time.Second
	bceConfig.Endpoint = cfg.Endpoint + "/internal-api"
	bceConfig.UserAgent = CceUserAgent + cfg.ClusterID
	cceClient := cce.NewClient(cce.NewConfig(bceConfig))
	cceClient.SetDebug(true)
	t.Log("begin !")
	ins, _ := cceClient.ListInstances(cfg.ClusterID)
	fmt.Printf("%+v\n", ins)

}
