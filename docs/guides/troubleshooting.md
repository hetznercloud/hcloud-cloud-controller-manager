# Troubleshooting

## Load Balancers

### Load Balancer Targets not Added

If your node is not added as a Load Balancer target, use the following snippet to check if your nodes are excluded from external Load Balancers.

```bash
kubectl get nodes --show-labels | grep node.kubernetes.io/exclude-from-external-load-balancers
```
