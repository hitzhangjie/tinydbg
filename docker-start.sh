#!/bin/bash -e

# debugger need priviledges including, ptrace, etc.
count=`docker ps -a | grep debugger.env | wc -l` 

if [ $count -eq 0 ]
then
    docker run -it --rm                                                         \
    -v `pwd -P`:/root/debugger101/dlv                                           \
    --name debugger.env --cap-add ALL                                           \
    --workdir /root/debugger101/dlv                                             \
    debugger.env                                                                \
    /bin/bash
else
    docker exec -it -w /root/debugger101/dlv debugger.env /bin/bash
fi

