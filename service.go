package main

import (
	"fmt"
	"time"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

func restartService(serviceName string, c chan<- string) (ok bool) {
	c <- "Подключение к Service Control Manager"
	m, err := mgr.Connect()
	if err != nil {
		c <- fmt.Sprintf("Не удалось подключиться к Service Control Manager: %v", err)
		return false
	}
	defer m.Disconnect()

	s, err := m.OpenService(serviceName)
	if err != nil {
		c <- fmt.Sprintf("Не удалось открыть сервис %s: %v", serviceName, err)
		return false
	}
	defer s.Close()

	status, err := s.Query()
	if err != nil {
		c <- fmt.Sprintf("Не удалось получить статус сервиса: %v", err)
		return false
	}
	if status.State != svc.Stopped {
		c <- "Остановка сервиса"
		status, err = s.Control(svc.Stop)
		if err != nil {
			c <- fmt.Sprintf("Не удалось остановить сервис: %v", err)
			return false
		}

		timeout := time.Now().Add(20 * time.Second)
		for status.State != svc.Stopped {
			if time.Now().After(timeout) {
				c <- "Превышено время ожидания остановки"
				return false
			}
			time.Sleep(500 * time.Millisecond)
			status, err = s.Query()
			if err != nil {
				c <- fmt.Sprintf("Не удалось получить статус сервиса: %v", err)
				return false
			}
		}
	}

	c <- "(Пере-)Запуск сервиса"
	err = s.Start()
	if err != nil {
		c <- fmt.Sprintf("Не удалось запустить сервис: %v", err)
		return false
	}

	timeout := time.Now().Add(20 * time.Second)
	for {
		status, err = s.Query()
		if err != nil {
			c <- fmt.Sprintf("Не удалось получить статус сервиса: %v", err)
			return false
		}
		if status.State == svc.Running {
			c <- "Сервис запущен"
			return true
		}
		if time.Now().After(timeout) {
			c <- "превышено время ожидания запуска"
			return false
		}
		time.Sleep(500 * time.Millisecond)
	}
}
