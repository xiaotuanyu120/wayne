package openapi

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/Qihoo360/wayne/src/backend/client"
	"github.com/Qihoo360/wayne/src/backend/models"
	"github.com/Qihoo360/wayne/src/backend/models/response"
	"github.com/Qihoo360/wayne/src/backend/resources/pod"
	"github.com/Qihoo360/wayne/src/backend/util/logs"
)

// An array of the pod.
// swagger:response resppodlist
type resppodlist struct {
	// in: body
	// Required: true
	Body struct {
		response.ResponseBase
		Pods []response.Pod `json:"pods"`
	}
}

// swagger:parameters PodInfoParam
type PodInfoParam struct {
	// A label key of k8s pod.
	// in: query
	// Required: true
	LabelSelector string `json:"labelSelector"`
	// Required: true
	Cluster string `json:"cluster"`
}

// swagger:parameters PodInfoFromIPParam
type PodInfoFromIPParam struct {
	// A list of ip.
	// in: query
	// Required: true
	IPS string          `json:"ips"`
	ips map[string]bool `json:"-"`
	// Required: true
	Cluster string `json:"cluster"`
}

// swagger:route GET /get_pod_info pod PodInfoParam
//
// 用于获取线上所有 pod 中包含请求条件中 labelSelector 指定的特定 label 的 pod
//
// 返回 每个 pod 的 pod IP 和 所有 label 列表。
// 需要绑定全局 apikey 使用。该接口的权限控制为只能使用全局 apikey 的原因是查询条件为 labelSelector ，是对所有 app 的 条件过滤。
//
//     Responses:
//       200: resppodlist
//       401: responseState
//       500: responseState
// @router /get_pod_info [get]
func (c *OpenAPIController) GetPodInfo() {
	if !c.CheckoutRoutePermission(GetPodInfoAction) {
		return
	}
	if c.APIKey.Type != models.GlobalAPIKey {
		c.AddErrorAndResponse("You can only use global APIKey in this action!", http.StatusUnauthorized)
		return
	}

	clis := client.Clients()
	podList := resppodlist{}
	podList.Body.Code = http.StatusOK
	params := PodInfoParam{c.GetString("labelSelector"), c.GetString("cluster")}
	if params.Cluster == "" {
		c.AddErrorAndResponse("Invalid cluster parameter:must required!", http.StatusBadRequest)
		return
	}
	if clis[params.Cluster] == nil {
		c.AddErrorAndResponse("Invalid cluster parameter:not exist!", http.StatusBadRequest)
		return
	}
	cli := clis[params.Cluster]
	pods, err := pod.GetAllPodByLabelSelector(cli, params.LabelSelector)
	if err != nil {
		logs.Error(fmt.Sprintf("Failed to parse metadata: %s", err.Error()))
		c.AddErrorAndResponse(fmt.Sprintf("Maybe a problematic k8s cluster(%s)!", params.Cluster), http.StatusInternalServerError)
		return
	}
	for _, p := range pods {
		podList.Body.Pods = append(podList.Body.Pods, response.Pod{Labels: p.Labels, PodIp: p.PodIp})
	}
	c.HandleResponse(podList.Body)
}

// swagger:route GET /get_pod_info_from_ip pod PodInfoFromIPParam
//
// 用于通过线上 kubernetes Pod IP 反查对应 Pod 信息的接口
//
// 返回 每个 pod 的 pod IP 和 所有 label 列表。
// 需要绑定全局 apikey 使用。该接口的权限控制为只能使用全局 apikey 的原因是查询条件为 IP ，是对所有 app 的 条件过滤。
//
//     Responses:
//       200: resppodlist
//       401: responseState
//       500: responseState
// @router /get_pod_info_from_ip [get]
func (c *OpenAPIController) GetPodInfoFromIP() {
	if !c.CheckoutRoutePermission(GetPodInfoFromIPAction) {
		return
	}
	if c.APIKey.Type != models.GlobalAPIKey {
		c.AddErrorAndResponse("You can only use global APIKey in this action!", http.StatusUnauthorized)
		return
	}
	params := PodInfoFromIPParam{IPS: c.GetString("ips"), Cluster: c.GetString("cluster")}
	if params.Cluster == "" {
		c.AddErrorAndResponse("Invalid cluster parameter:must required!", http.StatusBadRequest)
		return
	}
	params.ips = make(map[string]bool)
	for _, ip := range strings.Split(params.IPS, ",") {
		params.ips[ip] = true
	}
	manager, err := client.Manager(params.Cluster)
	if err != nil {
		c.AddErrorAndResponse("Invalid cluster parameter:not exist!", http.StatusBadRequest)
		return
	}
	pods := pod.GetPodsBySelectorFromCache(manager.Indexer, "", nil)
	if err != nil {
		logs.Error(fmt.Sprintf("Failed to parse metadata: %s", err.Error()))
		c.AddErrorAndResponse(fmt.Sprintf("Maybe a problematic k8s cluster(%s)!", params.Cluster), http.StatusInternalServerError)
		return
	}
	podList := resppodlist{}
	podList.Body.Code = http.StatusOK
	for _, p := range pods {
		if params.ips[p.Status.PodIP] {
			podList.Body.Pods = append(podList.Body.Pods, response.Pod{Labels: p.Labels, PodIp: p.Status.PodIP})
		}
	}
	c.HandleResponse(podList.Body)

}
