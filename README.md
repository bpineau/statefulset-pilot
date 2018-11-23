# statefulset-pilot

/!\ WARNING this is EXPERIMENTAL

**statefulset-pilot** calls validating hooks between each pod update during a statefulset rollout.

Hooks can return an error to temporarily postpone the next pod update, they'll be called again later.

Hooks are provided the previous and next pods structs: they can also execute pre and post update tasks.


## Using the statefulset-pilot

Statefulsets are subscribed to a pilot hook with the `dd-statefulset-pilot: hook-name` label.

For instance, to register a statefulset to the `elasticsearch` statefulset-pilot hook,
we would set the `dd-statefulset-pilot: elasticsearch` label.

```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: es-cluster
  labels:
    # subscribes this sts to the statefulset-pilot's "elasticsearch" hook
    dd-statefulset-pilot: elasticsearch
spec:
  replicas: 3
  updateStrategy:
    rollingUpdate:
      # We set partition == replicas by default, so rollouts will wait for the controller
      partition: 3
    type: RollingUpdate
```

Pod without this `dd-statefulset-pilot` label are ignored by statefulset-pilot controller.

You can remove the label at any time (including during a rollout) to unregister from the
statefulset-pilot. The pilot can safely be restarted at any time.


## Writing hooks

Hooks are Go code implementing the following interface:

```Go
// STSRolloutHooks is called between statefulset pods updates.
// If the hook returns an error, it will be called again later
// until it returns nil; then the next pod is updated.
type STSRolloutHooks interface {
	// Name returns the hook's name
	Name() string

	// PodUpdateTransition is called between pods updates.
	// prev is the previously updated pod (or nil when we're starting a new rollout).
	// next is the pod we're about to update (or nil after we updated the last pod).
	// If PodUpdateTransition returns an error, the controller will postpone the update,
	// and will call PodUpdateTransition again later, until it succeed.
	// If PodUpdateTransition returns nil, the controller will proceed updating the next pod.
	PodUpdateTransition(prev, next *v1.Pod) error
}
```

They must then be registered from factory.go `init()`:
```Go
  import "github.com/bpineau/statefulset-pilot/pkg/hooks/myhook"
  ...
  Register("myhook", myhook.New)
```

