package clusterservice

//Client Client for handling extra resources cleanup for a cluster
type Client interface {
	//DeleteResources delete resources belonging to a cluster based on filters from additional tags
	DeleteResourcesForCluster(clusterId string, tags map[string]string, dryRun bool) (*Report, error)
}
