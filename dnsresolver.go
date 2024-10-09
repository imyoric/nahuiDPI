package main

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"
)

var DNSCache sync.Map

// Set добавляет или обновляет IP-адрес в кэше.
func Set(key string, value net.IP) {
	DNSCache.Store(key, value)
}

// Get извлекает IP-адрес из кэша по ключу.
func Get(key string) (net.IP, bool) {
	value, ok := DNSCache.Load(key)
	if !ok {
		return nil, false
	}
	return value.(net.IP), true
}

// Delete удаляет запись из кэша по ключу.
func Delete(key string) {
	DNSCache.Delete(key)
}

func GetPreferredIP(dnsServer string, domain string) (net.IP, error) {
	ip := net.ParseIP(domain).To4()

	if ip.To16() != nil {
		return ip, nil
	}

	if ip.To4() != nil {
		return ip, nil
	}

	r, e := Get(domain)
	if e {
		return r, nil
	}

	// Создаем кастомный DNS Resolver с заданным DNS-сервером
	resolver := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return net.DialTimeout("udp", dnsServer, 2*time.Second)
		},
	}

	// Запрашиваем записи для домена
	ipAddresses, err := resolver.LookupIPAddr(context.Background(), domain)
	if err != nil {
		return nil, err
	}

	// Ищем первую AAAA запись (IPv6), затем A запись (IPv4)
	for _, ip := range ipAddresses {
		var toRet net.IP = nil

		if ip.IP.To16() != nil { // Проверяем на IPv6
			toRet = ip.IP
		}

		if ip.IP.To4() != nil { // Проверяем на IPv4
			toRet = ip.IP
		}

		if toRet != nil {
			Set(domain, ip.IP)
			return toRet, nil
		}
	}

	return nil, fmt.Errorf("no valid IP records found for domain %s", domain)
}
