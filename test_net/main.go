package main

import (
	"fmt"
	"strings"

	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
)

func main() {
	// Найдём процесс PathOfExile
	procs, _ := process.Processes()
	var poePid int32
	for _, p := range procs {
		name, _ := p.Name()
		if strings.Contains(strings.ToLower(name), "pathofexile") {
			poePid = p.Pid
			exe, _ := p.Exe()
			fmt.Printf("Найден процесс: %s (PID: %d)\n", exe, poePid)
			break
		}
	}

	if poePid == 0 {
		fmt.Println("Процесс PathOfExile не найден")
		return
	}

	// Получаем все сетевые подключения
	connections, err := net.Connections("all")
	if err != nil {
		fmt.Printf("Ошибка получения подключений: %v\n", err)
		return
	}

	fmt.Printf("\nВсего подключений: %d\n", len(connections))
	fmt.Println("\nПодключения PathOfExile:")
	fmt.Println("=" + strings.Repeat("=", 79))

	count := 0
	uniqueIPs := make(map[string]bool)

	for _, conn := range connections {
		if conn.Pid == poePid {
			count++
			remoteIP := conn.Raddr.IP
			remotePort := conn.Raddr.Port
			status := conn.Status

			if remoteIP != "" && remoteIP != "0.0.0.0" && remoteIP != "::" {
				uniqueIPs[remoteIP] = true
				connType := "TCP"
				if conn.Type == 2 {
					connType = "UDP"
				}
				fmt.Printf("%-5s %-20s:%-6d -> %-20s:%-6d %s\n",
					connType,
					conn.Laddr.IP, conn.Laddr.Port,
					remoteIP, remotePort,
					status)
			}
		}
	}

	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Printf("Найдено %d подключений\n", count)
	fmt.Printf("\nУникальные IP адреса (%d):\n", len(uniqueIPs))
	
	// Группируем по /8 подсетям
	subnets := make(map[string][]string)
	for ip := range uniqueIPs {
		parts := strings.Split(ip, ".")
		if len(parts) == 4 {
			subnet := parts[0] + ".0.0.0/8"
			subnets[subnet] = append(subnets[subnet], ip)
		}
	}

	fmt.Println("\nПодсети /8:")
	for subnet, ips := range subnets {
		fmt.Printf("  %s: %v\n", subnet, ips)
	}
}

