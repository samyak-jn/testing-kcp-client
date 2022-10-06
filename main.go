package main

import (
	"context"
	"flag"
	"fmt"
	"math/rand"

	// "os"
	"time"

	apisv1alpha1 "github.com/kcp-dev/kcp/pkg/apis/apis/v1alpha1"
	tenancyv1alpha1 "github.com/kcp-dev/kcp/pkg/apis/tenancy/v1alpha1"
	corev1 "k8s.io/api/core/v1"

	datav1alpha1 "github.com/kcp-dev/controller-runtime-example/api/v1alpha1"
	// "k8s.io/client-go/discovery"
	// ctrl "sigs.k8s.io/controller-runtime"

	kcpclienthelper "github.com/kcp-dev/apimachinery/pkg/client"
	"github.com/kcp-dev/logicalcluster/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

var workspaceName string

func parentWorkspace() logicalcluster.Name {
	flag.Parse()
	if workspaceName == "" {
		fmt.Println("--workspace cannot be empty")
	}

	return logicalcluster.New(workspaceName)
}

func loadClusterConfig(clusterName logicalcluster.Name) *rest.Config {

	restConfig, err := config.GetConfigWithContext("base")
	if err != nil {
		fmt.Println("failed to load *rest.Config: %v", err)
	}
	return rest.AddUserAgent(kcpclienthelper.ConfigWithCluster(restConfig, clusterName), "test-user")
}

func loadClient(clusterName logicalcluster.Name) client.Client {

	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		fmt.Println("failed to add client go to scheme: %v", err)
	}
	if err := tenancyv1alpha1.AddToScheme(scheme); err != nil {
		fmt.Println("failed to add %s to scheme: %v", tenancyv1alpha1.SchemeGroupVersion, err)
	}
	if err := datav1alpha1.AddToScheme(scheme); err != nil {
		fmt.Println("failed to add %s to scheme: %v", datav1alpha1.GroupVersion, err)
	}
	if err := apisv1alpha1.AddToScheme(scheme); err != nil {
		fmt.Println("failed to add %s to scheme: %v", apisv1alpha1.SchemeGroupVersion, err)
	}
	tenancyClient, err := client.New(loadClusterConfig(clusterName), client.Options{Scheme: scheme})
	if err != nil {
		fmt.Println("failed to create a client: %v", err)
	}
	return tenancyClient
}

func createlistWorkspace(clusterName logicalcluster.Name) client.Client {

	parent, ok := clusterName.Parent()
	if !ok {
		fmt.Println("cluster %s has no parent", clusterName)
	}
	c := loadClient(parent)
	fmt.Println("creating workspace %s", clusterName)

	// if err := c.List(context.TODO(), &tenancyv1alpha1.ClusterWorkspaceList{
	// 	TypeMeta: metav1.TypeMeta{
	// 		Kind:       "",
	// 		APIVersion: "",
	// 	},
	// 	ListMeta: metav1.ListMeta{
	// 		SelfLink:           "",
	// 		ResourceVersion:    "",
	// 		Continue:           "",
	// 		RemainingItemCount: new(int64),
	// 	},
	// 	Items: []tenancyv1alpha1.ClusterWorkspace{},
	// }); err != nil {
	// 	fmt.Println("failed to list workspace: %s: %v", clusterName, err)
	// 	return nil
	// }

	if err := c.Create(context.TODO(), &tenancyv1alpha1.ClusterWorkspace{
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterName.Base(),
		},
		Spec: tenancyv1alpha1.ClusterWorkspaceSpec{
			Type: tenancyv1alpha1.ClusterWorkspaceTypeReference{
				Name: "universal",
				Path: "root",
			},
		},
	}); err != nil {
		fmt.Errorf("failed to create workspace: %s: %v", clusterName, err)
	}

	fmt.Println("waiting for workspace %s to be ready", clusterName)
	var workspace tenancyv1alpha1.ClusterWorkspace
	if err := wait.PollImmediate(100*time.Millisecond, wait.ForeverTestTimeout, func() (done bool, err error) {
		fetchErr := c.Get(context.TODO(), client.ObjectKey{Name: clusterName.Base()}, &workspace)
		if fetchErr != nil {
			fmt.Println("failed to get workspace %s: %v", clusterName, err)
			return false, fetchErr
		}
		var reason string
		if actual, expected := workspace.Status.Phase, tenancyv1alpha1.ClusterWorkspacePhaseReady; actual != expected {
			reason = fmt.Sprintf("phase is %s, not %s", actual, expected)
			fmt.Println("not done waiting for workspace %s to be ready: %s", clusterName, reason)
		}
		return reason == "", nil
	}); err != nil {
		fmt.Errorf("workspace %s never ready: %v", clusterName, err)
	}

	return nil
}

const characters = "abcdefghijklmnopqrstuvwxyz"

func randomName() string {
	b := make([]byte, 10)
	for i := range b {
		b[i] = characters[rand.Intn(len(characters))]
	}
	return string(b)
}

func main() {
	fmt.Println("Exploring KCP controller runtime client")

	workspaceCluster := parentWorkspace()
	c := createlistWorkspace(workspaceCluster)

	namespaceName := randomName()
	fmt.Println("creating namespace %s|%s", workspaceCluster, namespaceName)
	if err := c.Create(context.TODO(), &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: namespaceName},
	}); err != nil {
		fmt.Println("failed to create a namespace: %v", err)
		return
	}

	// restConfig := ctrl.GetConfigOrDie()
	// discoveryClient, err := discovery.NewDiscoveryClientForConfig(restConfig)
	// if err != nil {
	// 	fmt.Println(err, "failed to create discovery client")
	// 	os.Exit(1)
	// }
	// apiGroupList, err := discoveryClient.ServerGroups()
	// if err != nil {
	// 	fmt.Println(err, "failed to get server groups")
	// 	os.Exit(1)
	// }

	// for _, group := range apiGroupList.Groups {
	// 	if group.Name == apisv1alpha1.SchemeGroupVersion.Group {
	// 		for _, version := range group.Versions {
	// 			if version.Version == apisv1alpha1.SchemeGroupVersion.Version {
	// 				fmt.Println(group)
	// 			}
	// 		}
	// 	}
	// }

}
