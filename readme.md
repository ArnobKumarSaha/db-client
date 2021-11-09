In this repo, I tried to work with mongoDB objects and its pods , by using In-Cluster method.

To do that, I had to make an image of my project and then run it as a job in k8s cluster .
Dockerfile, Makefile & Job template has been taken from https://github.com/faem/kubernetes-go-test.

Note::
- kubeconfig has to be "", as cluster er vitore kubeconfig nai.
- this job will be created on default namespace (not in demo)


I also add 

` - kind: ServiceAccount 
    name: default
    namespace: default
`

in the 'subjects' section by editing the clusterrolebinding named 'cluster-admin' to make the required permission.

