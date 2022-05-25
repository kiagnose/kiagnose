# Kiagnose Installation

In order to install Kiagnose on your cluster:

1. Clone this repository:

```bash
git clone https://github.com/kiagnose/kiagnose.git
```

2. Apply the following manifest from the project root directory (as a `cluster-admin` user):

```bash
kubectl apply -f ./manifests/kiagnose.yaml
```

This manifest contains the following objects:

- `kiagnose` [Namespace](https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces/) object.
- `kiagnose` [ServiceAccount](https://kubernetes.io/docs/reference/access-authn-authz/rbac/#service-account-permissions)
  object.
- `kiagnose` [ClusterRole](https://kubernetes.io/docs/reference/access-authn-authz/rbac/#role-and-clusterrole)
  and [ClusterRoleBinding](https://kubernetes.io/docs/reference/access-authn-authz/rbac/#rolebinding-and-clusterrolebinding)
  objects.

# Kiagnose Removal

1. Delete the objects contained in the following manifest from the project root directory (as a `cluster-admin` user):

```bash
kubectl delete -f ./manifests/kiagnose.yaml
```
