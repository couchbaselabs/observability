wget https://golang.org/dl/go1.16.5.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.16.5.linux-amd64.tar.gz
echo "export PATH=$PATH:/usr/local/go/bin" >> ~/.profile
source ~/.profile
go version
echo "export GOPATH=$HOME/go" >> ~/.profile
echo "export PATH=$PATH:$GOPATH/bin" >> ~/.profile
source ~/.profile
