# kaniko-image
kaniko image k8s

## Requirements
kaniko-secret
this is created by kubectl,just for your private registry auth.
you can execute docker login then you can get a config.json in HOME/.docker 

```bash
kubectl create namespace kaniko

kubectl create secret generic kaniko-secret --from-file=/root/.docker/config.json  -n kaniko

kubectl label node  $NODENAME  kaniko=enabled

kubectl taint node $NODENAME  kaniko=enabled:NoSchedule
```
## Usage
git pull your own repo, you must build a git image with ssh private key.and replace the default image alpine/git in the  source code.

```bash
go build 
```
 Usage of ./kaniko-image:
```
   -c  string
    	(optional) absolute path to the kubeconfig file (default "/root/.kube/config")
   -n string
     	namespace (default "default")
   -p string
    	projrct name
   -r string
        registry name
   -repo string
    	provide a repo of your project
```

### Example
```
kaniko-image -n kaniko -p spin-kub-demo -repo https://github.com/cuishuaigit/spin-kub-demo.git -r fastop/kaniko-demo:coding
```
