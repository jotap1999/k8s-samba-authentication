
# webhook service server 

## Install GO

```bash

wget https://dl.google.com/go/go1.14.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.14.linux-amd64.tar.gz
echo "PATH=$PATH:/usr/local/go/bin" >>~/.bashrc &&  . ~/.bashrc
```

```bash
#check
go version 
```

## Configure the WebHook service  
```bash
git clone https://github.com/jotap1999/k8s-samba-authentication.git
cd k8s-samba-authentication

go get github.com/go-ldap/ldap
go get k8s.io/api/authentication/v1
```


Edit file  main.go  with base on the LDAP/SAMBA configuration

nano main.go
```bash
#Line 18 - If the LDAP server is configured "over ssl/tls"
ldapURL = "ldaps://" + os.Args[1]

#Line 95
user :=  fmt.Sprintf("%s@KUBER.NET", username)  

#Line 104
"cn=Users,dc=kuber,dc=net"
```


## Compile the code 
```bash
GOOS=linux GOARCH=amd64 go build main.go
```

## Generate certificates
Create self-signed certificate. It is recomended to use a certificate signed by a CA 
```bash
openssl req -x509 -newkey rsa:2048 -nodes \
    -subj "/CN=localhost" \
    -keyout key.pem \
    -out cert.pem
```
## Run the WebHook service 
```bash
./main SERVER-LDAP  key.pem cert.pem  &>/var/log/k8s-samba-authentication.log &
```
## Test
nano testldap.json
```bash
  {
    "apiVersion": "authentication.k8s.io/v1",
    "kind": "TokenReview",
    "spec": {
      "token": "user:userpassword"
    }
  }
```
```bash
curl -k -X POST -d @testldap.json https://127.0.0.1
# If the status is empty the webhook is not working  
 "status": {
    "user": {}
  }

```
## Use systemd to iniciate on BOOT
nano /etc/systemd/system/webhook.service

```bash 
[Unit]
Description=Samba AD Webhook Authentication Server
After=network.target

[Service]
Type=simple
ExecStart=/root/k8s-samba-authentication/main 127.0.0.1 /root/k8s-samba-authentication/key.pem /root/k8s-samba-authentication/cert.pem

RestartSec=10
Restart=always

[Install]
WantedBy=multi-user.target
```

```bash 
systemctl start webhook.service
systemctl enable webhook.service
```

# Kubernetes server

```bash
#Install kubeadm and Docker
curl -o- https://raw.githubusercontent.com/jotap1999/k8s-docker-Install-Script-Ubuntu/master/install.sh  | bash
```

## Create the Webhook Token configuration file 
```bash
cat <<EOF > /root/webhook-config.yaml
apiVersion: v1
kind: Config
clusters:
  - name: authn
    cluster:
      server: https://X.X.X.X       #WebHook Server
      insecure-skip-tls-verify: true    #If the certificate isn't signed by a CA
users:
  - name: kube-apiserver
contexts:
- context:
    cluster: authn
    user: kube-apiserver
  name: authn
current-context: authn
EOF
```

## Create the kubeadm configuration file

```bash
cat <<EOF >kubeadm-config.yaml
apiVersion: kubeadm.k8s.io/v1beta2
kind: ClusterConfiguration
apiServer:
  extraVolumes:
    - name: authentication-token-webhook-config-file
      mountPath: /etc/webhook-config.yaml
      hostPath: /root/webhook-config.yaml   
  extraArgs:
    authentication-token-webhook-config-file: /etc/webhook-config.yaml
  certSANs:
    - X.X.X.X       #IP address Kubernetes API server listens
networking:
  podSubnet: "10.244.0.0/16"
EOF

```


## Convert the file into a more recent version and iniciate the Kubernetes with that same file
```bash
kubeadm config migrate --old-config kubeadm-config.yaml --new-config kubeadm-config-new.yaml
kubeadm init --config kubeadm-config-new.yaml 
```


```bash
#Install a CNI plugin. 
#Example the Flannel
kubectl apply -f https://raw.githubusercontent.com/coreos/flannel/master/Documentation/kube-flannel.yml
```

```bash
#Create the ClusterRole or Roles to the users or groups
#For example:

kind: ClusterRoleBinding
metadata:
  name: k8s-admin-group
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cluster-admin
subjects:
- kind: Group
  name: kuberadmin
  apiGroup: rbac.authorization.k8s.io
```


# Client 

```bash
kubectl config set-credentials testuser \
 --token user:userpassword

kubectl config set-context user-context \
--cluster=kubernetes --user=user

kubectl config use-context  user-context

kubectl config set-cluster kubernetes  \
  --insecure-skip-tls-verify=true  \
  --server https://X.X.X.X:6443 

```

# Reference
- https://github.com/weibeld/k8s-ldap-authentication
