package gather

import (
	"context"
	// _ "embed"
	"fmt"
	subctlversion "github.com/submariner-io/submariner-operator/pkg/version"
	"html/template"
	"io"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"os"
	"path/filepath"
)

// Embed the file content as string.
// //go:embed layout.html
//var layout string

var layout = `<!DOCTYPE html>
<html>
<style>
    th,
    td {
        padding: 2px;
    }
</style>
<body>
<h2 id="cluster-info"><a href="#{{.ClusterName}}-info">Displaying information for {{.ClusterName}}</a></h2>
<h3 id="versions"><a href="#{{.ClusterName}}-versions">Versions</a></h3>
<table>
    <tr>
        <td>subctl version:</td>
        <td>{{.Versions.Subctl}}</td>
    </tr>
    <tr>
        <td>Submariner version:</td>
        <td>{{.Versions.Subm}}</td>
    </tr>
    <tr>
        <td>Kubernetes Server version:</td>
        <td>{{.Versions.K8sServer}}</td>
    </tr>
</table>
<h3 id="cluster-config"><a href="#{{.ClusterName}}-cluster-config">Cluster configuration</a></h3>
<table>
    <tr>
        <td>CNI Plugin:</td>
        <td>{{.ClusterConfig.CNIPlugin}}</td>
    </tr>
    <tr>
        <td>Cloud Provider:</td>
        <td>{{.ClusterConfig.CloudProvider}}</td>
    </tr>
    <tr>
        <td>Total node(s)</td>
        <td>{{.ClusterConfig.TotalNode}}</td>
    </tr>
    <tr>
        <td>Master node(s)</td>
        <td>{{.ClusterConfig.MasterNodeNumber}}</td>
    </tr>
    {{range $name, $uuid := .ClusterConfig.MasterNode}}
    <tr>
        <td>&nbsp;</td>
        <td>{{$name}}: {{$uuid}}</td>
    </tr>
    {{end}}
    <tr>
        <td>Gateway node(s)</td>
        <td>{{.ClusterConfig.GWNodeNumber}}</td>
    </tr>
    {{range $name, $uuid := .ClusterConfig.GatewayNode}}
    <tr>
        <td>&nbsp;</td>
        <td>{{$name}}: {{$uuid}}</td>
    </tr>
    {{end}}
</table>
<h3 id="node-info"><a href="#{{.ClusterName}}-node-info">Node Information</a></h3>
<table style="table-layout:fixed; width:100%; text-align:left;">
    <tr>
        <th>Node name</th>
        <th>Operating System</th>
        <th>Container Runtime Version</th>
        <th>Kubelet Version</th>
        <th>KubeProxy Version</th>
    </tr>
    {{range .NodeConfig}}
    <tr>
        <td>{{.Name}}</td>
        <td style="width: 40%;">{{.Info.OperatingSystem}} {{.Info.OSImage}} {{.Info.KernelVersion}} {{.Info.Architecture}}</td>
        <td style="width: 25%;">{{.Info.ContainerRuntimeVersion}}</td>
        <td style="width: 15%;">{{.Info.KubeletVersion}}</td>
        <td style="width: 15%;">{{.Info.KubeProxyVersion}}</td>
    </tr>
    {{end}}
</table>
<h3 id="log-collected"><a href="#{{.ClusterName}}-log-collected">Summary of logs gathered</a></h3>
<table style="table-layout:fixed; width:100%; text-align:left;">
    <tr>
        <th>Pod name</th>
        <th>Namespace</th>
        <th>Pod state</th>
        <th>Restart count</th>
        <th>Log file(s)</th>
    </tr>
    {{range .PodLogs}}
    <tr>
        <td style="width: 10%;">{{.PodName}}</td>
        <td style="width: 10%;">{{.Namespace}}</td>
        <td style="width: 10%;">{{.PodState}}</td>
        {{ if gt .RestartCount 0 }}
        <td style="width: 10%; color: red">{{.RestartCount}}</td>
        {{ else }}
        <td style="width: 10%;">{{.RestartCount}}</td>
        {{end}}
        {{range $index, $filename := .LogFileName}}
        <td style="width: 60%;"><a href="{{$filename}}">{{$filename}}</a></td>
        {{end}}
    </tr>
    {{end}}
</table>
<h3 id="resources-collected"><a href="#{{.ClusterName}}-resources-collected">Summary of resources gathered</a></h3>
<table style="table-layout:fixed; width:100%; text-align:left;">
    <tr>
        <th>Resource name</th>
        <th>Namespace</th>
        <th>Resource type</th>
        <th>Resource file(s)</th>
    </tr>
    {{range .ResourceInfo}}
    <tr>
        <td>{{.Name}}</td>
        <td>{{.Namespace}}</td>
        <td>{{.Type}}</td>
        <td>{{.FileName}}</td>
    </tr>
    {{end}}
</table>
</body>
</html>
`

type version struct {
	Subctl    string
	Subm      string
	K8sServer string
}

type clusterConfig struct {
	CNIPlugin        string
	CloudProvider    string
	TotalNode        int
	GatewayNode      map[string]types.UID
	GWNodeNumber     int
	MasterNode       map[string]types.UID
	MasterNodeNumber int
}

type nodeConfig struct {
	Name string
	Info v1.NodeSystemInfo
}

type logInfo struct {
	PodName      string
	Namespace    string
	RestartCount int32
	PodState     v1.PodPhase
	LogFileName  []string
}

type resourceInfo struct {
    Name      string
    Namespace string
    Type      string
    FileName  string
}

type data struct {
	ClusterName   string
	Versions      version
    ClusterConfig clusterConfig
	NodeConfig    []nodeConfig
	PodLogs       []logInfo
	ResourceInfo  []resourceInfo
}

func ClusterInformation(info Info) {
	dataGathered := getClusterInfo(info)
	file := createFile(info.DirName)
	WriteToHTML(file, dataGathered)
}

func getClusterInfo(info Info) data {
	versions := getVersions(info)
	config := getClusterConfig(info)
	nConfig := getNodeConfig(info)
	podLogs := populatePodLogInfo()
	resourcesInfo := populateResourceInfo()

	d := data{
		ClusterName:      info.ClusterName,
		Versions:         versions,
	    ClusterConfig:    config,
	    NodeConfig: nConfig,
	    PodLogs: podLogs,
	    ResourceInfo: resourcesInfo,
	}
	return d
}

func getClusterConfig(info Info) clusterConfig {
	cniPlugin := "Not found"
	if info.Submariner != nil {
		cniPlugin = info.Submariner.Status.NetworkPlugin
	}
	gwNodes := getGWNodes(info)
	mNodes := getMasterNodes(info)
	allNodes, _ := listNodes(info, metav1.ListOptions{})
	config := clusterConfig{
		CNIPlugin:        cniPlugin,
		CloudProvider:    "AWS", // TODO
		TotalNode:        len(allNodes.Items),
		GatewayNode:      gwNodes,
		GWNodeNumber:     len(gwNodes),
		MasterNode:       mNodes,
		MasterNodeNumber: len(mNodes),
	}
	return config
}

func getVersions(info Info) version {
	k8sServerVersion, err := info.ClientSet.Discovery().ServerVersion()
	if err != nil {
		fmt.Println("error in getting k8s server version %s", err)
	}

	submVer := "Not installed"
	if info.Submariner != nil {
		submVer = info.Submariner.Spec.Version
	}

	Versions := version{
		Subctl:    subctlversion.Version,
		Subm:      submVer,
		K8sServer: k8sServerVersion.String(),
	}
	return Versions
}

func getSpecificNode(info Info, selector string) map[string]types.UID {
	var node = make(map[string]types.UID)
	nodes, err := listNodes(info, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		fmt.Println(err)
	}
	for _, n := range nodes.Items {
		node[n.GetName()] = n.GetUID()
	}
	return node
}

func getGWNodes(info Info) map[string]types.UID {
	selector := labels.SelectorFromSet(labels.Set(map[string]string{"submariner.io/gateway": "true"}))
	return getSpecificNode(info, selector.String())
}

func getMasterNodes(info Info) map[string]types.UID {
	selector := "node-role.kubernetes.io/master="
	return getSpecificNode(info, selector)
}

func getNodeConfig(info Info) []nodeConfig {
	nodes, err := listNodes(info, metav1.ListOptions{})
	if err != nil {
		fmt.Println(err)
	}

	var nodeConfigs []nodeConfig
	for _, allNode := range nodes.Items {
		nodeInfo := v1.NodeSystemInfo{
			KernelVersion:           allNode.Status.NodeInfo.KernelVersion,
			OSImage:                 allNode.Status.NodeInfo.OSImage,
			ContainerRuntimeVersion: allNode.Status.NodeInfo.ContainerRuntimeVersion,
			KubeletVersion:          allNode.Status.NodeInfo.KubeletVersion,
			KubeProxyVersion:        allNode.Status.NodeInfo.KubeProxyVersion,
			OperatingSystem:         allNode.Status.NodeInfo.OperatingSystem,
			Architecture:            allNode.Status.NodeInfo.Architecture,
		}
		name := allNode.GetName()
		config := nodeConfig{
			Name: name,
			Info: nodeInfo,
		}
		nodeConfigs = append(nodeConfigs, config)
	}
    return nodeConfigs
}

func listNodes(info Info, listOptions metav1.ListOptions ) (*v1.NodeList, error) {
	nodes, err := info.ClientSet.CoreV1().Nodes().List(context.TODO(), listOptions)
	if err != nil {
		fmt.Println("error listing nodes")
	}
	return nodes, nil
}

func createFile(dirname string) io.Writer {
	fileName := filepath.Join(dirname, "tally.html")
	f, err := os.OpenFile(fileName, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		fmt.Sprintf("Error creating file %s", fileName)
	}
	return f
}
func WriteToHTML(fileWriter io.Writer, cData data) {
	t := template.Must(template.New("layout.html").Parse(layout))
	err := t.Execute(fileWriter, cData)
	if err != nil {
		fmt.Println(err)
	}
}
