# go-send
A Pure Go implementation for P2P File Transfer

# Project-Goals

* A CLI which can do the following
  
  -> $ go-send -token <unique_token> -src </path/of/file> -mode S
  
  -> $ go-send -token <unique_token> -dest </path/of/dir/> -mode R
  
* A Signalling server that can connect between many go-send clients
