# -*- mode: ruby -*-
# vi: set ft=ruby :

VAGRANTFILE_API_VERSION = "2"

Vagrant.configure(VAGRANTFILE_API_VERSION) do |config|
    config.vm.box = "cass-dock"
    config.vm.box_url = "http://cloud-images.ubuntu.com/vagrant/precise/current/precise-server-cloudimg-amd64-vagrant-disk1.box"
    config.vm.network :private_network, ip: "192.168.50.100"

    # plugin conflict
    # if Vagrant.has_plugin?("vagrant-vbguest") then
    #     config.vbguest.auto_update = false
    # end

    config.vm.provider "virtualbox" do |v|
        v.name = "cass-dock"
        v.customize ["modifyvm", :id, "--memory", 2048]
    end

    config.vm.provision "shell",
        inline: $set_limits

    config.vm.provision "docker" do |d|
        d.pull_images "zmarcantel/cassandra"
    end

    config.vm.provision "shell",
        inline: $docker_limits

    config.vm.provision "docker" do |d|
        d.run "seed", auto_assign_name: false,
          args: "--name cass0",
          image: "zmarcantel/cassandra"

        d.run "first", auto_assign_name: false,
          args: "--name cass1 --link cass0:cass0",
          image: "zmarcantel/cassandra"

        d.run "second", auto_assign_name: false,
          args: "--name cass2 --link cass0:cass0 --link cass1:cass1",
          image: "zmarcantel/cassandra"
    end

    config.vm.provision "shell",
        inline: "sudo apt-get install -y mercurial bzr"

    config.vm.provision "shell",
        inline: $install_go

    config.vm.provision "shell",
        inline: $run_tests
    # sudo bin/cmm -p `for i in $(sudo docker ps -q); do sudo docker inspect $i | grep IPA; done | xargs echo | sed 's/ /, /g'
end

$set_limits = <<SCRIPT
ulimit -n 200000
ulimit -l unlimited
SCRIPT

$docker_limits = <<SCRIPT
sed -i '/stop on.*/a limit nofile 200000 200000' /etc/init/docker.conf
sed -i '/stop on.*/a limit memlock unlimited unlimited' /etc/init/docker.conf
SCRIPT


$install_go = <<SCRIPT
wget -q -nc https://go.googlecode.com/files/go1.2.1.linux-amd64.tar.gz
tar -xzf go1.2.1.linux-amd64.tar.gz
rm -rf /usr/local/bin/go && mv go /usr/local/bin/
mkdir -p /usr/lib/go
SCRIPT

$run_tests = <<SCRIPT
cd /vagrant
export GOROOT=/usr/local/bin/go
export GOPATH=/usr/lib/go
export PATH=$PATH:$GOROOT/bin
make dependencies
export DOCKER_IPS=`bash test/docker_ips.sh`
TEST_HOSTS=$DOCKER_IPS go test
SCRIPT
