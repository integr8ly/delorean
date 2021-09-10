package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi"
)

func convertClusterTagsToAWSTagFilter(clusterId string, additionalTags map[string]string) []*resourcegroupstaggingapi.TagFilter {
	tagFilters := []*resourcegroupstaggingapi.TagFilter{
		{
			Key:    aws.String(tagKeyClusterId),
			Values: aws.StringSlice([]string{clusterId}),
		},
	}
	for tagKey, tagVal := range additionalTags {
		tagFilters = append(tagFilters, &resourcegroupstaggingapi.TagFilter{
			Key:    aws.String(tagKey),
			Values: aws.StringSlice([]string{tagVal}),
		})
	}
	return tagFilters
}
