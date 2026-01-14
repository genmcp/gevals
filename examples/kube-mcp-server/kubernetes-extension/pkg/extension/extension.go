package extension

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/genmcp/gevals/pkg/extension/sdk"
	"github.com/google/jsonschema-go/jsonschema"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
)

// Extension wraps the SDK extension with Kubernetes client
type Extension struct {
	*sdk.Extension
	client dynamic.Interface
}

// New creates a new Kubernetes extension
func New() *Extension {
	ext := &Extension{}
	ext.Extension = sdk.NewExtension(
		sdk.ExtensionInfo{
			Name:        "kubernetes",
			Version:     "0.1.0",
			Description: "Kubernetes resource operations using client-go",
		},
		sdk.WithInitializeHandler(ext.handleInitialize),
	)

	ext.registerOperations()
	return ext
}

func (e *Extension) handleInitialize(config map[string]any) error {
	kubeconfigPath := ""

	if path, ok := config["kubeconfig"].(string); ok {
		kubeconfigPath = path
	}

	// Expand ~ to home directory
	if strings.HasPrefix(kubeconfigPath, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		kubeconfigPath = filepath.Join(home, kubeconfigPath[1:])
	}

	// If no kubeconfig specified, use default
	if kubeconfigPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		kubeconfigPath = filepath.Join(home, ".kube", "config")
	}

	kubeconfig, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return fmt.Errorf("failed to build kubeconfig from %s: %w", kubeconfigPath, err)
	}

	client, err := dynamic.NewForConfig(kubeconfig)
	if err != nil {
		return fmt.Errorf("failed to create dynamic client: %w", err)
	}

	e.client = client
	return nil
}

func (e *Extension) registerOperations() {
	e.AddOperation(
		sdk.NewOperation("create",
			sdk.WithDescription("Create a Kubernetes resource"),
			sdk.WithParams(jsonschema.Schema{
				Type:        "object",
				Description: "Kubernetes resource spec (apiVersion, kind, metadata, spec, etc.)",
				Properties: map[string]*jsonschema.Schema{
					"apiVersion": {
						Type:        "string",
						Description: "API version (e.g., v1, apps/v1)",
					},
					"kind": {
						Type:        "string",
						Description: "Resource kind (e.g., Pod, Namespace, Deployment)",
					},
					"metadata": {
						Type:        "object",
						Description: "Resource metadata (name, namespace, labels, annotations)",
					},
					"spec": {
						Type:        "object",
						Description: "Resource spec (optional, depends on resource type)",
					},
				},
				Required: []string{"apiVersion", "kind", "metadata"},
			}),
		),
		e.handleCreate,
	)

	e.AddOperation(
		sdk.NewOperation("wait",
			sdk.WithDescription("Wait for a condition on a Kubernetes resource"),
			sdk.WithParams(jsonschema.Schema{
				Type:        "object",
				Description: "Resource reference with condition to wait for",
				Properties: map[string]*jsonschema.Schema{
					"apiVersion": {
						Type:        "string",
						Description: "API version (e.g., v1, apps/v1)",
					},
					"kind": {
						Type:        "string",
						Description: "Resource kind (e.g., Pod, Deployment)",
					},
					"metadata": {
						Type:        "object",
						Description: "Resource metadata (name, namespace)",
					},
					"condition": {
						Type:        "string",
						Description: "Condition type to wait for (e.g., Ready, Available)",
					},
					"status": {
						Type:        "string",
						Description: "Expected condition status (default: True)",
					},
					"timeout": {
						Type:        "string",
						Description: "Timeout duration (e.g., 60s, 5m, default: 60s)",
					},
				},
				Required: []string{"apiVersion", "kind", "metadata", "condition"},
			}),
		),
		e.handleWait,
	)

	e.AddOperation(
		sdk.NewOperation("delete",
			sdk.WithDescription("Delete a Kubernetes resource"),
			sdk.WithParams(jsonschema.Schema{
				Type:        "object",
				Description: "Resource reference to delete",
				Properties: map[string]*jsonschema.Schema{
					"apiVersion": {
						Type:        "string",
						Description: "API version (e.g., v1, apps/v1)",
					},
					"kind": {
						Type:        "string",
						Description: "Resource kind (e.g., Pod, Namespace)",
					},
					"metadata": {
						Type:        "object",
						Description: "Resource metadata (name, namespace)",
					},
				},
				Required: []string{"apiVersion", "kind", "metadata"},
			}),
		),
		e.handleDelete,
	)
}

func (e *Extension) handleCreate(ctx context.Context, req *sdk.OperationRequest) (*sdk.OperationResult, error) {
	if e.client == nil {
		return sdk.Failure(fmt.Errorf("kubernetes client not initialized")), nil
	}

	// Args is the resource spec as a map
	resourceSpec, ok := req.Args.(map[string]any)
	if !ok {
		return sdk.Failure(fmt.Errorf("args must be a resource spec object")), nil
	}

	obj := &unstructured.Unstructured{Object: resourceSpec}

	gvk := obj.GroupVersionKind()
	if gvk.Kind == "" {
		return sdk.Failure(fmt.Errorf("kind is required")), nil
	}

	gvr := gvkToGVR(gvk)

	var result *unstructured.Unstructured
	var err error
	namespace := obj.GetNamespace()

	if namespace != "" {
		result, err = e.client.Resource(gvr).Namespace(namespace).Create(ctx, obj, metav1.CreateOptions{})
	} else {
		result, err = e.client.Resource(gvr).Create(ctx, obj, metav1.CreateOptions{})
	}

	if err != nil {
		return sdk.Failure(fmt.Errorf("failed to create resource: %w", err)), nil
	}

	return sdk.SuccessWithOutputs(
		fmt.Sprintf("Created %s/%s", gvk.Kind, result.GetName()),
		map[string]string{
			"name":            result.GetName(),
			"namespace":       result.GetNamespace(),
			"uid":             string(result.GetUID()),
			"resourceVersion": result.GetResourceVersion(),
		},
	), nil
}

// resourceRef extracts resource reference info from args
type resourceRef struct {
	apiVersion string
	kind       string
	name       string
	namespace  string
}

func parseResourceRef(args map[string]any) (*resourceRef, error) {
	apiVersion, _ := args["apiVersion"].(string)
	kind, _ := args["kind"].(string)

	if apiVersion == "" {
		return nil, fmt.Errorf("apiVersion is required")
	}
	if kind == "" {
		return nil, fmt.Errorf("kind is required")
	}

	ref := &resourceRef{
		apiVersion: apiVersion,
		kind:       kind,
	}

	if metadata, ok := args["metadata"].(map[string]any); ok {
		ref.name, _ = metadata["name"].(string)
		ref.namespace, _ = metadata["namespace"].(string)
	}

	if ref.name == "" {
		return nil, fmt.Errorf("metadata.name is required")
	}

	return ref, nil
}

func (r *resourceRef) gvr() (schema.GroupVersionResource, error) {
	gv, err := schema.ParseGroupVersion(r.apiVersion)
	if err != nil {
		return schema.GroupVersionResource{}, fmt.Errorf("invalid apiVersion: %w", err)
	}
	return schema.GroupVersionResource{
		Group:    gv.Group,
		Version:  gv.Version,
		Resource: kindToResource(r.kind),
	}, nil
}

func (e *Extension) handleWait(ctx context.Context, req *sdk.OperationRequest) (*sdk.OperationResult, error) {
	if e.client == nil {
		return sdk.Failure(fmt.Errorf("kubernetes client not initialized")), nil
	}

	args, ok := req.Args.(map[string]any)
	if !ok {
		return sdk.Failure(fmt.Errorf("args must be an object")), nil
	}

	ref, err := parseResourceRef(args)
	if err != nil {
		return sdk.Failure(err), nil
	}

	condition, _ := args["condition"].(string)
	if condition == "" {
		return sdk.Failure(fmt.Errorf("condition is required")), nil
	}

	status, _ := args["status"].(string)
	if status == "" {
		status = "True"
	}

	timeoutStr, _ := args["timeout"].(string)
	if timeoutStr == "" {
		timeoutStr = "60s"
	}

	timeout, err := time.ParseDuration(timeoutStr)
	if err != nil {
		return sdk.Failure(fmt.Errorf("invalid timeout format: %w", err)), nil
	}

	gvr, err := ref.gvr()
	if err != nil {
		return sdk.Failure(err), nil
	}

	var lastStatus string
	err = wait.PollUntilContextTimeout(ctx, time.Second, timeout, true, func(ctx context.Context) (bool, error) {
		var obj *unstructured.Unstructured
		var getErr error

		if ref.namespace != "" {
			obj, getErr = e.client.Resource(gvr).Namespace(ref.namespace).Get(ctx, ref.name, metav1.GetOptions{})
		} else {
			obj, getErr = e.client.Resource(gvr).Get(ctx, ref.name, metav1.GetOptions{})
		}

		if getErr != nil {
			return false, nil // Keep polling on transient errors
		}

		conditions, found, err := unstructured.NestedSlice(obj.Object, "status", "conditions")
		if err != nil || !found {
			lastStatus = "NoConditions"
			return false, nil
		}

		for _, c := range conditions {
			cond, ok := c.(map[string]any)
			if !ok {
				continue
			}

			condType, _, _ := unstructured.NestedString(cond, "type")
			condStatus, _, _ := unstructured.NestedString(cond, "status")

			if condType == condition {
				lastStatus = condStatus
				return condStatus == status, nil
			}
		}

		lastStatus = "ConditionNotFound"
		return false, nil
	})

	if err != nil {
		return sdk.FailureWithMessage(
			fmt.Sprintf("Condition %s=%s not met", condition, status),
			fmt.Errorf("timed out waiting for %s/%s: last status was %s", ref.kind, ref.name, lastStatus),
		), nil
	}

	return sdk.Success(fmt.Sprintf("%s/%s condition %s=%s", ref.kind, ref.name, condition, status)), nil
}

func (e *Extension) handleDelete(ctx context.Context, req *sdk.OperationRequest) (*sdk.OperationResult, error) {
	if e.client == nil {
		return sdk.Failure(fmt.Errorf("kubernetes client not initialized")), nil
	}

	args, ok := req.Args.(map[string]any)
	if !ok {
		return sdk.Failure(fmt.Errorf("args must be an object")), nil
	}

	ref, err := parseResourceRef(args)
	if err != nil {
		return sdk.Failure(err), nil
	}

	gvr, err := ref.gvr()
	if err != nil {
		return sdk.Failure(err), nil
	}

	propagation := metav1.DeletePropagationForeground
	deleteOpts := metav1.DeleteOptions{
		PropagationPolicy: &propagation,
	}

	if ref.namespace != "" {
		err = e.client.Resource(gvr).Namespace(ref.namespace).Delete(ctx, ref.name, deleteOpts)
	} else {
		err = e.client.Resource(gvr).Delete(ctx, ref.name, deleteOpts)
	}

	if err != nil {
		return sdk.Failure(fmt.Errorf("failed to delete resource: %w", err)), nil
	}

	return sdk.Success(fmt.Sprintf("Deleted %s/%s", ref.kind, ref.name)), nil
}

// gvkToGVR converts a GroupVersionKind to GroupVersionResource
func gvkToGVR(gvk schema.GroupVersionKind) schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    gvk.Group,
		Version:  gvk.Version,
		Resource: kindToResource(gvk.Kind),
	}
}

// kindToResource converts a Kind to its plural resource name
func kindToResource(kind string) string {
	lower := kind
	if len(lower) > 0 {
		lower = string(lower[0]|32) + lower[1:]
	}

	// Handle common irregular plurals
	switch lower {
	case "ingress":
		return "ingresses"
	case "networkPolicy":
		return "networkpolicies"
	default:
		return lower + "s"
	}
}

// Run starts the extension, listening for JSON-RPC messages on stdin/stdout
func (e *Extension) Run(ctx context.Context) error {
	return e.Extension.Run(ctx)
}
