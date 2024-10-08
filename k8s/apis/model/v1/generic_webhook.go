package v1

import (
	"context"
	"fmt"
	"gopls-workspace/utils/diagnostic"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/eclipse-symphony/symphony/k8s/apis/metrics/v1"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type GetSubResourceNums func() (n int, err error)

var cacheClient client.Client
var readerClient client.Reader
var commoncontainermetrics *metrics.Metrics

func InitCommonContainerWebHook(mgr ctrl.Manager) error {
	if commoncontainermetrics == nil {
		initmetrics, err := metrics.New()
		if err != nil {
			return err
		}
		commoncontainermetrics = initmetrics
	}
	cacheClient = mgr.GetClient()
	readerClient = mgr.GetAPIReader()
	return nil
}

func SetupWebhookWithManager(mgr ctrl.Manager, resource client.Object) error {
	mgr.GetFieldIndexer().IndexField(context.Background(), resource, ".metadata.name", func(rawObj client.Object) []string {
		return []string{rawObj.GetName()}
	})

	return ctrl.NewWebhookManagedBy(mgr).
		For(resource).
		Complete()
}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func DefaultImpl(log logr.Logger, ctx context.Context, r client.Object) {
	diagnostic.InfoWithCtx(log, ctx, "default", "name", r.GetName(), "kind", r.GetObjectKind())
}

func ValidateCreateImpl(log logr.Logger, ctx context.Context, r client.Object) (admission.Warnings, error) {
	diagnostic.InfoWithCtx(log, ctx, "validate create", "name", r.GetName(), "kind", r.GetObjectKind())
	return nil, nil
}
func ValidateUpdateImpl(log logr.Logger, ctx context.Context, r client.Object, old runtime.Object) (admission.Warnings, error) {
	diagnostic.InfoWithCtx(log, ctx, "validate update", "name", r.GetName(), "kind", r.GetObjectKind())
	return nil, nil
}

func ValidateDeleteImpl(log logr.Logger, ctx context.Context, r client.Object, getSubResourceNums GetSubResourceNums) (admission.Warnings, error) {

	diagnostic.InfoWithCtx(log, ctx, "validate delete", "name", r.GetName(), "kind", r.GetObjectKind())

	validateDeleteTime := time.Now()
	validationError := validateDeleteContainerImpl(log, ctx, r, getSubResourceNums)
	if validationError != nil {
		commoncontainermetrics.ControllerValidationLatency(
			validateDeleteTime,
			metrics.CreateOperationType,
			metrics.InvalidResource,
			r.GetObjectKind().GroupVersionKind().Kind)
	} else {
		commoncontainermetrics.ControllerValidationLatency(
			validateDeleteTime,
			metrics.CreateOperationType,
			metrics.ValidResource,
			r.GetObjectKind().GroupVersionKind().Kind)
	}

	return nil, validationError
}

func validateDeleteContainerImpl(log logr.Logger, ctx context.Context, r client.Object, getSubResourceNums GetSubResourceNums) error {
	itemsNum, err := getSubResourceNums()
	if err != nil {
		diagnostic.ErrorWithCtx(log, ctx, err, "could not list nested resources", "name", r.GetName(), "namespace", r.GetNamespace(), "kind", r.GetObjectKind())
		return apierrors.NewBadRequest(fmt.Sprintf("%s could not list nested resources for %s.", r.GetObjectKind(), r.GetName()))
	}
	if itemsNum > 0 {
		diagnostic.ErrorWithCtx(log, ctx, err, "nested resources are not empty", "name", r.GetName(), "namespace", r.GetNamespace(), "kind", r.GetObjectKind())
		return apierrors.NewBadRequest(fmt.Sprintf("%s nested resources with root resource '%s' are not empty", r.GetObjectKind(), r.GetName()))
	}

	return nil
}
