package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

// criando 3 canais
var (
	saida     = make(chan string)
	entrada   = make(chan string)
	erroCanal = make(chan error)
)

/* obs.: as funçoes vao virar go routines - gerenciando partes da conexao */
func leInput() {
	fmt.Println("Olá! Para melhor aproveitamento do sistema digite '/opcoes' por favor! ")
	for {
		reader := bufio.NewReader(os.Stdin)
		m, err := reader.ReadString('\n')
		if err != nil {
			panic(err)
		}
		entrada <- m
	}
}

func leConn(conn net.Conn) {
	for {
		buf := make([]byte, 1024)
		n, err := conn.Read(buf)
		if err != nil {
			erroCanal <- err
			return
		}
		m := string(buf[:n])
		saida <- m
	}
}

func envia(conn net.Conn, m string) {
	if conn == nil {
		return
	}
	_, err := conn.Write([]byte(m))
	if err != nil {
		fmt.Println(err)
		conn.Close()
		conn = nil
	}
}

func main() {

	var conn net.Conn
	go leInput() //go routine que cuida dos input

	sinais := make(chan os.Signal, 1)
	signal.Notify(sinais, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case <-sinais:
			fmt.Println("\nDesconectando...")
			os.Exit(0)
		case m := <-saida:
			if m == "ping\n" {
				envia(conn, "pong\n")
				continue
			}
			fmt.Print(m)
		case m := <-entrada:
			cmd := strings.Split(strings.TrimSpace(m), " ")
			command := cmd[0]

			switch command {
			case "/opcoes":
				o := "/opcoes                   - mostre as opcoes\n"
				o += "/sair                     - sair\n"
				o += "/conecte <host:porta>     - conectar com o servidor"
				fmt.Println(o)
				envia(conn, "/opcoes\n") // envia opcoes to server
				continue
			case "/conecte":
				server := cmd[1]
				var err error
				fmt.Printf("Conectando %s...\n", server)
				conn, err = net.Dial("tcp", server)
				if err != nil {
					fmt.Printf("Erro: %s\n", err)
					continue
				}
				fmt.Printf("Conectado em %s\n", server)
				go leConn(conn)
				println("Digite novamente '/opcoes' para visualizar mais do sistema!")
				continue
			case "/sair":
				fmt.Println("\nDesconectando..")
				os.Exit(0)
			case "":
				continue
			}
			if conn == nil {
				fmt.Println("Não conectado.")
				continue
			}
			envia(conn, m)
		}
	}
}
