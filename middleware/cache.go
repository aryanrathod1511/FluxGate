package middleware

import "net/http"

func CheckCache(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//check in the store
		//data, ok := checkCache(r)

		// if ok {
		// 	writeCachedResponse(w, data)
		// 	return
		// }

		//rw = NewResponseWriter(w)
		//next.ServeHTTP(rw, r)
		//make new entry
		//save entry in store
		//rw.FlushTo(w)

	})
}
