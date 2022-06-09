echo "Creating Docker Image - Python"
docker build --progress=plain -f ./build/dockerfiles/DockerfilePython -t virtual_machine_python .

echo "Creating Docker Image - Node"
docker build --progress=plain -f ./build/dockerfiles/DockerfileNode -t virtual_machine_node .

echo "Creating Docker Image - Rust"
docker build --progress=plain -f ./build/dockerfiles/DockerfileRust -t virtual_machine_rs .

#echo "Creating Docker Image - CSharp"
#docker build --progress=plain -f ./build/dockerfiles/DockerFileCSharp -t virtual_machine_cs .

echo "Retrieving Installed Docker Images"
docker images
