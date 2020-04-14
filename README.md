# Go-Seele-Sub
Seele subchain (beta version) was officially released on March 29, 2020. For more details, please see [Seele Stem subchain protocol](https://medium.com/@SeeleTech/seele-stem-subchain-protocol-b5eceb02aaa3). So far, Practical Byzantine Fault Tolerance (PBFT) is used as block consensus protocol.

# Download (without building)
If you want to directly run the node and use client without setting up the compiling enviroment and building the executable files, you can choose right version to download and run:

| Operation System |      Download Link     |
|---------|----------------------------------------------------------|
| Linux   | [https://github.com/seeleteam/go-seele-sub/releases]|
| MacOs   | [https://github.com/seeleteam/go-seele-sub/releases]|
| Windows | [https://github.com/seeleteam/go-seele-sub/releases]|

# Or Download & Build the source

Building the Seele project requires both a Go (version 1.7 or later) compiler and a C compiler. You can install them using your favourite package manager. Once the dependencies are installed, run

- Building the Seele project requires both a Go (version 1.7 or later) compiler and a C compiler. Install Go v1.10 or higher, Git, and the C compiler.

- Clone the go-seele-sub repository to the GOPATH directory:

```
go get -u -v github.com/seeleteam/go-seele-sub/...
```

- Change the cloned folder name from "go-seele-sub" to "go-seele"

- Once successfully cloned code and rename the folder:

```
cd GOPATH/src/github.com/seeleteam/go-seele/
```

- Linux & Mac

```
make all
```

- Windows
```
buildall.bat
```

# How to Start Subchain 

- Creat A Subchain (Creator)

  1. Create A Subchain Smart Contract and Deploy on Seele MainNet. Pleae refer to [SubChain Smart Contract](https://seeletech.gitbook.io/wiki/developer/intro/subchain_contract)
  2. Releaes and Anounce(for example, [mall_pay_template](https://github.com/seeleteam/go-seele-sub/projects))

- Run Subchain (Operators & Normal Users)
  
  3. For running a subchain node, please refer to [Subchain Start Guide](https://seeletech.gitbook.io/wiki/developer/intro/subchain_start)
  
  4. User controller middleware to operate subchain: 
        - (Operations: Relay/Deposit/Challenge/Exit ), please refer to [Controller User Guide](https://seeletech.gitbook.io/wiki/developer/intro/seele-anchor-cli/0-user) 
        - [Controller Test Guide](https://seeletech.gitbook.io/wiki/developer/intro/seele-anchor-cli/1-test) 
        -  [Controller Configuration](https://seeletech.gitbook.io/wiki/developer/intro/seele-anchor-cli/2-conf) 
  
For more usage details and deeper explanations, please go to [Seele Subchain Introduction](https://seeletech.gitbook.io/wiki/developer/intro)  and [Seele Wiki](https://seeletech.gitbook.io/wiki/)([Older version](https://seeleteam.github.io/seele-doc/index.html)).

# Contribution

Thank you for considering helping out with our source code. We appreciate any contributions, even the smallest fixes.

Here are some guidelines before you start:
* Code must adhere to the official Go [formatting](https://golang.org/doc/effective_go.html#formatting) guidelines (i.e. uses [gofmt](https://golang.org/cmd/gofmt/)).
* Pull requests need to be based on and opened against the `master` branch.
* We use reviewable.io as our review tool for any pull request. Please submit and follow up on your comments in this tool. After you submit a PR, there will be a `Reviewable` button in your PR. Click this button, it will take you to the review page (it may ask you to login).
* If you have any questions, feel free to join [chat room](https://gitter.im/seeleteamchat/dev) to communicate with our core team.


# License

[go-seele/LICENSE](https://github.com/seeleteam/go-seele/blob/master/LICENSE)
