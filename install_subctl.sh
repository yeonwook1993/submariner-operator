set -e

go mod tidy
make bin/subctl
mkdir ~/.local/bin
export PATH=$PATH:~/.local/bin
echo export PATH=\$PATH:~/.local/bin >> ~/.profile
cp bin/subctl ~/.local/bin/subctl
