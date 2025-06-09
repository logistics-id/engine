package rest

import (
	"fmt"

	"github.com/gorilla/mux"
)

func debugRoutes(r *mux.Router) {
	fmt.Println("\nREGISTERED ROUTES:")
	fmt.Println("-------------------------------------------------------------")
	fmt.Printf("%-8s | %-25s | %-30s\n", "METHOD", "PATH", "HANDLER")
	fmt.Println("-------------------------------------------------------------")

	_ = r.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		path, _ := route.GetPathTemplate()
		methods, _ := route.GetMethods()
		name := route.GetName()

		for _, m := range methods {
			fmt.Printf("%-8s | %-25s | %-30s\n", m, path, name)
		}
		return nil
	})

	fmt.Println("-------------------------------------------------------------")
}
