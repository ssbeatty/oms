// Copyright 2021 The Libc Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package libc // import "modernc.org/libc"

import (
	"modernc.org/libc/sys/types"
)

func X__libc_thr_yield(tls *TLS) int32 {
	panic(todo(""))
}

// int pthread_create(pthread_t *thread, const pthread_attr_t *attr, void *(*start_routine) (void *), void *arg);
func X__libc_create(tls *TLS, thread, attr, start_routine, arg uintptr) int32 {
	panic(todo(""))
}

func Xpthread_create(tls *TLS, thread, attr, start_routine, arg uintptr) int32 {
	panic(todo(""))
}

// int pthread_detach(pthread_t thread);
func X__libc_detach(tls *TLS, thread types.Pthread_t) int32 {
	panic(todo(""))
}

func Xpthread_detach(tls *TLS, thread types.Pthread_t) int32 {
	panic(todo(""))
}

// int pthread_mutex_lock(pthread_mutex_t *mutex);
func X__libc_mutex_lock(tls *TLS, mutex uintptr) int32 {
	panic(todo(""))
}

func Xpthread_mutex_lock(tls *TLS, mutex uintptr) int32 {
	panic(todo(""))
}

// int pthread_cond_signal(pthread_cond_t *cond);
func X__libc_cond_signal(tls *TLS, cond uintptr) int32 {
	panic(todo(""))
}

func Xpthread_cond_signal(tls *TLS, cond uintptr) int32 {
	panic(todo(""))
}

// int pthread_mutex_unlock(pthread_mutex_t *mutex);
func X__libc_mutex_unlock(tls *TLS, mutex uintptr) int32 {
	panic(todo(""))
}

func Xpthread_mutex_unlock(tls *TLS, mutex uintptr) int32 {
	panic(todo(""))
}

// int pthread_mutex_init(pthread_mutex_t *restrict mutex, const pthread_mutexattr_t *restrict attr);
func X__libc_mutex_init(tls *TLS, mutex, attr uintptr) int32 {
	panic(todo(""))
}

func Xpthread_mutex_init(tls *TLS, mutex, attr uintptr) int32 {
	panic(todo(""))
}

// int pthread_cond_init(pthread_cond_t *restrict cond, const pthread_condattr_t *restrict attr);
func X__libc_cond_init(tls *TLS, cond, attr uintptr) int32 {
	panic(todo(""))
}

func Xpthread_cond_init(tls *TLS, cond, attr uintptr) int32 {
	panic(todo(""))
}

// int pthread_cond_wait(pthread_cond_t *restrict cond, pthread_mutex_t *restrict mutex);
func X__libc_cond_wait(tls *TLS, cond, mutex uintptr) int32 {
	panic(todo(""))
}

func Xpthread_cond_wait(tls *TLS, cond, mutex uintptr) int32 {
	panic(todo(""))
}

// int pthread_cond_destroy(pthread_cond_t *cond);
func X__libc_cond_destroy(tls *TLS, cond uintptr) int32 {
	panic(todo(""))
}

func Xpthread_cond_destroy(tls *TLS, cond uintptr) int32 {
	panic(todo(""))
}

// int pthread_mutex_destroy(pthread_mutex_t *mutex);
func X__libc_mutex_destroy(tls *TLS, mutex uintptr) int32 {
	panic(todo(""))
}

func Xpthread_mutex_destroy(tls *TLS, mutex uintptr) int32 {
	panic(todo(""))
}

// int pthread_mutex_trylock(pthread_mutex_t *mutex);
func X__libc_mutex_trylock(tls *TLS, mutex uintptr) int32 {
	panic(todo(""))
}

func Xpthread_mutex_trylock(tls *TLS, mutex uintptr) int32 {
	panic(todo(""))
}

// int pthread_cond_broadcast(pthread_cond_t *cond);
func X__libc_cond_broadcast(tls *TLS, cond uintptr) int32 {
	panic(todo(""))
}

func Xpthread_cond_broadcast(tls *TLS, cond uintptr) int32 {
	panic(todo(""))
}
