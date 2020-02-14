## go-ssh

Package ssh provides tooling to manage ssh config.
Easy to use ssh without remember password or address.

--- 

## build

```bash
    $ ./build.sh
```

## install
```bash
    $ mkdir ~/.go-ssh && \
        cp ./bin/s ~/.go-ssh && \
        echo "export PATH=\$PATH:~/.go-ssh" >> ~/.bash_profile && \
        source ~/.bash_profile
```