############################################################
# Dockerfile to build sandbox for executing user code
# Based on Ubuntu
############################################################

FROM ubuntu:22.10

# Update the repository sources list
RUN echo "deb http://archive.ubuntu.com/ubuntu trusty main universe" > /etc/apt/sources.list

RUN apt-get update
RUN apt-get install -y curl

RUN apt-get install -y sudo
RUN apt-get install -y bc

#RUN apt-get upgrade
#Install all the languages/compilers we are supporting.
RUN apt-get install -y python
RUN apt-get install -y npm
RUN apt-get install -y nodejs


RUN mkdir -p /opt/rust && \
    curl https://sh.rustup.rs -sSf | HOME=/opt/rust sh -s -- --no-modify-path -y && \
    chmod -R 777 /opt/rust



