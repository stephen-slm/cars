echo "Creating Docker Image - Python"

docker build --progress=plain -f ./build/dockerfiles/DockerfilePython -t virtual_machine_python .

#echo "Creating Docker Image - Node"
#docker build -t 'virtual_machine_node' - < DockerFileNode
#
#echo "Creating Docker Image - CSharp"
#docker build -t 'virtual_machine_cs' - < DockerFileCSharp


echo "Retrieving Installed Docker Images"
docker images
