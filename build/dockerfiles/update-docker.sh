echo "Creating Docker Image - Python"
docker build -t 'virtual_machine_python' - < DockerFilePython

echo "Creating Docker Image - Node"
docker build -t 'virtual_machine_node' - < DockerFileNode

echo "Creating Docker Image - CSharp"
docker build -t 'virtual_machine_cs' - < DockerFileCSharp


echo "Retrieving Installed Docker Images"
docker images
