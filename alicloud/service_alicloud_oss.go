package alicloud

import (
	"strconv"
	"time"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/aliyun/terraform-provider-alicloud/alicloud/connectivity"
)

// OssService *connectivity.AliyunClient
type OssService struct {
	client *connectivity.AliyunClient
}

func (s *OssService) DescribeOssBucket(id string) (response oss.GetBucketInfoResult, err error) {
	request := map[string]string{"bucketName": id}
	var requestInfo *oss.Client
	raw, err := s.client.WithOssClient(func(ossClient *oss.Client) (interface{}, error) {
		requestInfo = ossClient
		return ossClient.GetBucketInfo(request["bucketName"])
	})
	if err != nil {
		if ossNotFoundError(err) {
			return response, WrapErrorf(err, NotFoundMsg, AliyunOssGoSdk)
		}
		return response, WrapErrorf(err, DefaultErrorMsg, id, "GetBucketInfo", AliyunOssGoSdk)
	}

	addDebug("GetBucketInfo", raw, requestInfo, request)
	response, _ = raw.(oss.GetBucketInfoResult)
	return
}

func (s *OssService) DescribeOssBucketReplication(id string) (response oss.GetBucketReplicationResult, err error) {
	request := map[string]string{"bucketName": id}
	var requestInfo *oss.Client

	raw, err := s.client.WithOssClient(func(ossClient *oss.Client) (interface{}, error) {
		requestInfo = ossClient
		return ossClient.GetBucketReplication(request["bucketName"])
	})
	if err != nil {
		// sometimes replication_rule is nil, only need to create bucket, so catch this error.
		if NoSuchReplicationConfigurationError(err) {
			return raw.(oss.GetBucketReplicationResult), nil
		}
		return response, WrapErrorf(err, DefaultErrorMsg, id, "GetBucketReplication", AliyunOssGoSdk)
	}

	addDebug("GetBucketReplication", raw, requestInfo, request)
	response, _ = raw.(oss.GetBucketReplicationResult)
	return
}

func (s *OssService) WaitForOssBucket(id string, status Status, timeout int) error {
	deadline := time.Now().Add(time.Duration(timeout) * time.Second)
	for {
		object, err := s.DescribeOssBucket(id)
		if err != nil {
			if NoSuchBucketError(err) {
				if status == Deleted {
					return nil
				}
			} else if AccessDeniedError(err) {
				// After you delete a bucket, get the bucket will return '404 NoSuchBucket'; But if the bucket was set replication_rule, oss will return '403 AccessDenied' in about 10min
				if status == Deleted {
					return nil
				}
			} else {
				return WrapError(err)
			}
		}

		if object.BucketInfo.Name != "" && status != Deleted {
			return nil
		}
		if time.Now().After(deadline) {
			return WrapErrorf(err, WaitTimeoutMsg, id, GetFunc(1), timeout, object.BucketInfo.Name, status, ProviderERROR)
		}
	}
}

func (s *OssService) WaitForOssBucketObject(bucket *oss.Bucket, id string, status Status, timeout int) error {
	deadline := time.Now().Add(time.Duration(timeout) * time.Second)
	for {
		exist, err := bucket.IsObjectExist(id)
		if err != nil {
			return WrapErrorf(err, DefaultErrorMsg, id, "IsObjectExist", AliyunOssGoSdk)
		}
		addDebug("IsObjectExist", exist)

		if !exist {
			return nil
		}

		if time.Now().After(deadline) {
			return WrapErrorf(err, WaitTimeoutMsg, id, GetFunc(1), timeout, strconv.FormatBool(exist), status, ProviderERROR)
		}
	}
}
