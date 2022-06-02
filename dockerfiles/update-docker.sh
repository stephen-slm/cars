# echo "Creating Docker Image"
# docker build -t 'virtual_machine' - < Dockerfile

echo "Creating Docker Image - Python"
docker build -t 'virtual_machine_python' - < DockerFilePython

echo "Creating Docker Image - Node"
docker build -t 'virtual_machine_node' - < DockerFileNode

echo "Retrieving Installed Docker Images"
docker images
