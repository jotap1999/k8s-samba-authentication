#############
##webhook service server 
##############
wget https://dl.google.com/go/go1.14.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.14.linux-amd64.tar.gz
echo "PATH=$PATH:/usr/local/go/bin" >>~/.bashrc &&  . ~/.bashrc

#check
go version 

git clone https://github.com/weibeld/k8s-ldap-authentication.git
cd k8s-ldap-authentication

go get github.com/go-ldap/ldap
go get k8s.io/api/authentication/v1


#editar ficheiro  main.go  com base na configuação do ldap 
#Correr o comando abaixo para obter as informações necessárias:
#SambaAD:
##ldbsearch -H ldap://SERVER-LDAP  -U Administrator --password=Password '(&(sAMAccountName=Administrator))' | grep distinguishedName
#OpenLDAP
##slapcat | grep modifiersName | head -1
  #Line 18 - Se o servidor LDAP estiver configurado "over ssl/tls"
    ldapURL = "ldaps://" + os.Args[1]
  #Line 95
      user :=  fmt.Sprintf("%s@KUBER.NET", username)  
  #Line 104
    "cn=Users,dc=kuber,dc=net"

GOOS=linux GOARCH=amd64 go build main.go

#Criar self-signed certificate. É recomendado usar um certificado assinado por uma CA 
openssl req -x509 -newkey rsa:2048 -nodes \
    -subj "/CN=localhost" \
    -keyout key.pem \
    -out cert.pem

./main SERVER-LDAP  key.pem cert.pem  &>/var/log/k8s-ldap-authentication.log &

#Test
nano testldap.json
  {
    "apiVersion": "authentication.k8s.io/v1",
    "kind": "TokenReview",
    "spec": {
      "token": "user:userpassword"
    }
  }

curl -k -X POST -d @testldap.json https://127.0.0.1
# Se o status estiver vazio, o webhook não está a funcionar  
 "status": {
    "user": {}
  }

##############
##Kubernetes server
##############
# Install kubeadm and Docker


cat <<EOF > /root/webhook-config.yaml
apiVersion: v1
kind: Config
clusters:
  - name: authn
    cluster:
      server: https://X.X.X.X       #WebHook Server
      insecure-skip-tls-verify: true    #Se o certificado não estiver assinado
users:
  - name: kube-apiserver
contexts:
- context:
    cluster: authn
    user: kube-apiserver
  name: authn
current-context: authn
EOF

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
  podSubnet: "10.244.0.0/16" #se usar o Flannel
EOF



kubeadm config migrate --old-config kubeadm-config.yaml --new-config kubeadm-config-new.yaml
kubeadm init --config kubeadm-config-new.yaml 


kubectl apply -f https://raw.githubusercontent.com/coreos/flannel/master/Documentation/kube-flannel.yml

#Criar as ClusterRole para os utilizadores ou grupos

##############
##Client 
##############
kubectl config set-credentials testuser \
 --token user:userpassword

kubectl config set-context user-context \
--cluster=kubernetes --user=user

kubectl config use-context  user-context

kubectl config set-cluster kubernetes  \
  --insecure-skip-tls-verify=true  \      
  --server https://192.168.114.130:6443 