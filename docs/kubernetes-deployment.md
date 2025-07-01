# Kubernetes 部署指南

## ☸️ 架构概述

在Kubernetes中，我们使用以下部署模式：
- **Server**: Deployment (可水平扩展)
- **Agent**: DaemonSet (每个节点一个实例)

## 🚀 快速部署

### 1. 创建命名空间

```bash
kubectl apply -f https://raw.githubusercontent.com/zhoushoujian/go-net-monitoring/main/k8s/namespace.yaml
```

### 2. 部署Server

```bash
kubectl apply -f https://raw.githubusercontent.com/zhoushoujian/go-net-monitoring/main/k8s/server-deployment.yaml
```

### 3. 部署Agent

```bash
kubectl apply -f https://raw.githubusercontent.com/zhoushoujian/go-net-monitoring/main/k8s/agent-daemonset.yaml
```

### 4. 验证部署

```bash
# 检查Pod状态
kubectl get pods -n monitoring

# 检查服务状态
kubectl get svc -n monitoring

# 查看日志
kubectl logs -n monitoring deployment/netmon-server
kubectl logs -n monitoring daemonset/netmon-agent
```

## 📋 详细配置

### Server Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: netmon-server
  namespace: monitoring
spec:
  replicas: 1  # 可根据需要调整
  selector:
    matchLabels:
      app: netmon-server
  template:
    spec:
      containers:
      - name: server
        image: zhoushoujian/go-net-monitoring:latest
        env:
        - name: COMPONENT
          value: "server"
        - name: SERVER_HOST
          value: "0.0.0.0"
        - name: SERVER_PORT
          value: "8080"
        ports:
        - containerPort: 8080
        resources:
          requests:
            memory: "128Mi"
            cpu: "100m"
          limits:
            memory: "512Mi"
            cpu: "500m"
```

### Agent DaemonSet

```yaml
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: netmon-agent
  namespace: monitoring
spec:
  selector:
    matchLabels:
      app: netmon-agent
  template:
    spec:
      hostNetwork: true  # 访问主机网络
      hostPID: true      # 访问主机进程
      containers:
      - name: agent
        image: zhoushoujian/go-net-monitoring:latest
        env:
        - name: COMPONENT
          value: "agent"
        - name: SERVER_URL
          value: "http://netmon-server.monitoring.svc.cluster.local:8080/api/v1/metrics"
        securityContext:
          privileged: true  # 需要特权模式
        resources:
          requests:
            memory: "64Mi"
            cpu: "50m"
          limits:
            memory: "256Mi"
            cpu: "200m"
```

## 🔧 配置管理

### 1. 使用ConfigMap

```bash
# 创建ConfigMap
kubectl create configmap netmon-config \
  --from-file=configs/agent.yaml \
  --from-file=configs/server.yaml \
  -n monitoring

# 在Pod中使用
spec:
  containers:
  - name: server
    volumeMounts:
    - name: config
      mountPath: /app/configs
  volumes:
  - name: config
    configMap:
      name: netmon-config
```

### 2. 使用Secret

```bash
# 创建Secret (如果有敏感配置)
kubectl create secret generic netmon-secret \
  --from-literal=api-key=your-api-key \
  -n monitoring

# 在Pod中使用
spec:
  containers:
  - name: server
    env:
    - name: API_KEY
      valueFrom:
        secretKeyRef:
          name: netmon-secret
          key: api-key
```

## 🌐 服务暴露

### 1. ClusterIP (集群内访问)

```yaml
apiVersion: v1
kind: Service
metadata:
  name: netmon-server
  namespace: monitoring
spec:
  type: ClusterIP
  ports:
  - port: 8080
    targetPort: 8080
  selector:
    app: netmon-server
```

### 2. NodePort (外部访问)

```yaml
apiVersion: v1
kind: Service
metadata:
  name: netmon-server-nodeport
  namespace: monitoring
spec:
  type: NodePort
  ports:
  - port: 8080
    targetPort: 8080
    nodePort: 30080
  selector:
    app: netmon-server
```

### 3. Ingress (域名访问)

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: netmon-ingress
  namespace: monitoring
spec:
  rules:
  - host: netmon.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: netmon-server
            port:
              number: 8080
```

## 📊 监控集成

### 1. Prometheus ServiceMonitor

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: netmon-server
  namespace: monitoring
spec:
  selector:
    matchLabels:
      app: netmon-server
  endpoints:
  - port: http
    path: /metrics
    interval: 30s
```

### 2. Grafana Dashboard

```bash
# 导入Dashboard ConfigMap
kubectl create configmap grafana-dashboard \
  --from-file=examples/grafana-dashboard.json \
  -n monitoring
```

## 🔒 安全配置

### 1. RBAC权限

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: netmon-agent
  namespace: monitoring

---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: netmon-agent
rules:
- apiGroups: [""]
  resources: ["nodes", "pods"]
  verbs: ["get", "list", "watch"]

---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: netmon-agent
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: netmon-agent
subjects:
- kind: ServiceAccount
  name: netmon-agent
  namespace: monitoring
```

### 2. Pod Security Policy

```yaml
apiVersion: policy/v1beta1
kind: PodSecurityPolicy
metadata:
  name: netmon-agent-psp
spec:
  privileged: true
  allowPrivilegeEscalation: true
  allowedCapabilities:
  - NET_ADMIN
  - NET_RAW
  - SYS_ADMIN
  hostNetwork: true
  hostPID: true
  runAsUser:
    rule: RunAsAny
  fsGroup:
    rule: RunAsAny
```

## 🔍 故障排除

### 1. Pod状态检查

```bash
# 查看Pod状态
kubectl get pods -n monitoring -o wide

# 查看Pod详细信息
kubectl describe pod <pod-name> -n monitoring

# 查看Pod日志
kubectl logs <pod-name> -n monitoring -f
```

### 2. 网络连通性

```bash
# 测试Service连通性
kubectl run test-pod --image=busybox --rm -it -- /bin/sh
wget -O- http://netmon-server.monitoring.svc.cluster.local:8080/health

# 测试DNS解析
nslookup netmon-server.monitoring.svc.cluster.local
```

### 3. 权限问题

```bash
# 检查ServiceAccount
kubectl get sa -n monitoring

# 检查RBAC权限
kubectl auth can-i get nodes --as=system:serviceaccount:monitoring:netmon-agent

# 检查Pod安全上下文
kubectl get pod <pod-name> -n monitoring -o yaml | grep -A 10 securityContext
```

## 📈 扩展和优化

### 1. 水平扩展Server

```bash
# 扩展Server副本数
kubectl scale deployment netmon-server --replicas=3 -n monitoring

# 配置HPA (水平Pod自动扩展)
kubectl autoscale deployment netmon-server \
  --cpu-percent=70 \
  --min=1 \
  --max=5 \
  -n monitoring
```

### 2. 资源限制优化

```yaml
resources:
  requests:
    memory: "128Mi"
    cpu: "100m"
  limits:
    memory: "512Mi"
    cpu: "500m"
```

### 3. 节点选择器

```yaml
spec:
  nodeSelector:
    kubernetes.io/os: linux
    node-type: monitoring
  tolerations:
  - key: node-role.kubernetes.io/master
    operator: Exists
    effect: NoSchedule
```

## 🔄 更新和维护

### 1. 滚动更新

```bash
# 更新镜像
kubectl set image deployment/netmon-server \
  server=zhoushoujian/go-net-monitoring:v1.1.0 \
  -n monitoring

# 查看更新状态
kubectl rollout status deployment/netmon-server -n monitoring

# 回滚更新
kubectl rollout undo deployment/netmon-server -n monitoring
```

### 2. 配置更新

```bash
# 更新ConfigMap
kubectl create configmap netmon-config \
  --from-file=configs/ \
  --dry-run=client -o yaml | kubectl apply -f -

# 重启Pod以应用新配置
kubectl rollout restart deployment/netmon-server -n monitoring
kubectl rollout restart daemonset/netmon-agent -n monitoring
```

### 3. 备份和恢复

```bash
# 备份配置
kubectl get all,configmap,secret -n monitoring -o yaml > netmon-backup.yaml

# 恢复配置
kubectl apply -f netmon-backup.yaml
```

## 🗑️ 清理资源

```bash
# 删除所有资源
kubectl delete namespace monitoring

# 或者逐个删除
kubectl delete -f k8s/agent-daemonset.yaml
kubectl delete -f k8s/server-deployment.yaml
kubectl delete -f k8s/namespace.yaml
```
