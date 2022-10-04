package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

type client struct {
	nome string
	conn net.Conn
}

var (
	clientes = make(map[string]*client, 0)
	mu       sync.Mutex
)

func deleteCliente(ip string) {
	mu.Lock()
	defer mu.Unlock()
	delete(clientes, ip)
}

func addCliente(ip string, c *client) {
	mu.Lock()
	defer mu.Unlock()
	clientes[ip] = c
}

// canal de leitura - leitura do que vem pela conexão
func handleLeConn(conn net.Conn, msgReadCh chan string, errCh chan error) {
	for {
		if msgReadCh == nil || conn == nil {
			return
		}
		buf := make([]byte, 1024)
		n, err := conn.Read(buf)
		if err != nil {
			errCh <- err
			return
		}
		m := string(buf[:n])
		msgReadCh <- m
	}
}

// envio de dados sendo feito diretamente pela conexao
func envia(conn net.Conn, msg string) bool {
	_, err := conn.Write([]byte(msg))
	if err != nil {
		if err == io.EOF {
			fmt.Printf("%v Conexão fechada\n", conn.RemoteAddr())
			return false
		}
		fmt.Printf("%v Erro: %v\n", conn.RemoteAddr(), err)
		return false
	}
	return true
}

func handler(conn net.Conn) { //handler principal que contra a conexao com cliente
	//instancias de canais
	pingInterval := time.Second * 5
	maxPingInterval := time.Second * 15
	msgReadCh := make(chan string)
	errCh := make(chan error)
	lastMsgTime := time.Now()

	clientInstance := &client{
		conn: conn,
		nome: "desconhecido:" + conn.RemoteAddr().String(),
	}

	addCliente(
		conn.RemoteAddr().String(),
		clientInstance)

	defer func() {
		close(msgReadCh)
		close(errCh)
		conn.Close()
		deleteCliente(conn.RemoteAddr().String())
	}()

	//go routine
	go handleLeConn(conn, msgReadCh, errCh)

	for {
		select {
		case <-time.After(pingInterval):
			if time.Since(lastMsgTime) > pingInterval {
				if !envia(conn, "ping\n") {
					return
				}
			}
			if time.Since(lastMsgTime) > maxPingInterval {
				fmt.Println("Conexão inativa, fechando.")
				return
			}
		case msg := <-msgReadCh:
			lastMsgTime = time.Now()
			cmd := strings.Split(strings.TrimSpace(msg), " ")

			command := cmd[0]

			//comandos que o cliente vai inserir no terminal
			switch command {
			case "pong":
				continue
			case "/nome":
				mu.Lock()
				c := clientes[conn.RemoteAddr().String()]
				oldNick := c.nome
				c.nome = cmd[1]
				clientes[conn.RemoteAddr().String()] = c
				msg := fmt.Sprintf("%s mudou o nome para %s\n", oldNick, cmd[1])
				for _, c := range clientes {
					if !envia(c.conn, msg) {
						mu.Unlock()
						return
					}
				}
				mu.Unlock()
				continue
			case "/conectados":
				mu.Lock()
				for _, c := range clientes {
					if !envia(conn, fmt.Sprintf("%s\n", c.nome)) {
						mu.Unlock()
						return
					}
				}
				mu.Unlock()

				continue
			case "/opcoes":
				o := "/conectados               - mostra todos os clientes conectados\n"
				o += "/nome <seu nome>          - informe seu nome\n"
				o += "/msg <nome> <mensagem>    - envia mensagem para outro cliente\n"
				if !envia(conn, o) {
					return
				}
				continue
			case "/msg":
				mu.Lock()
				for _, c := range clientes {
					if c.nome == cmd[1] {
						msg := fmt.Sprintf("%v: %v\n",
							clientInstance.nome,
							strings.Join(cmd[2:], " "))
						if !envia(c.conn, msg) {
							mu.Unlock()
							return
						}
						break
					}
				}
				mu.Unlock()

				continue
			case "":
				continue
			}

			mu.Lock()
			for _, c := range clientes {
				msg := fmt.Sprintf("%v: %v\n",
					clientInstance.nome, msg)
				if !envia(c.conn, msg) {
					mu.Unlock()
					return
				}
			}
			mu.Unlock()

		case err := <-errCh:
			if err == io.EOF {
				fmt.Printf("%v Conexão fechada\n", conn.RemoteAddr())
				return
			}
			fmt.Println("Erro ao ler a conexão", err)
			return
		}
	}
}

func main() {
	fmt.Println("Ouvindo na porta 8080")

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigs
		fmt.Println("\nDesligando servidor...")
		os.Exit(0)
	}()

	ln, _ := net.Listen("tcp", ":8080")

	for {
		conn, _ := ln.Accept()
		fmt.Printf("%v Conexão aceita\n", conn.RemoteAddr())
		//abrindo uma go routine c/ a func handler - passando a conexão
		go handler(conn)
	}
}
