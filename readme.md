Create docker image usable with kubernetes on machine without docker


Following command creates image from contents of src_dir to out.tar with tag testx:1
```
./gen src_dir out.tar testx:1
```

On machine with docker then:
```
docker load <out.tar
```

To launch pod with local docker image:
```
kubectl run -i --tty fun --image=testx:1 --restart=Never --image-pull-policy=Never -- /hello
```
