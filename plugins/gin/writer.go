package gin

import (
	"bufio"
	"net"
	"net/http"

	"github.com/gin-gonic/gin"
)

var _ (gin.ResponseWriter) = (*ginWriterDecorator)(nil)

type ginWriterDecorator struct {
	CustomWriter gin.ResponseWriter
}

func (g *ginWriterDecorator) Header() http.Header {
	return g.CustomWriter.Header()
}
func (g *ginWriterDecorator) WriteHeader(code int) {
	g.CustomWriter.WriteHeader(code)
}
func (g *ginWriterDecorator) Write(b []byte) (int, error) {
	return g.CustomWriter.Write(b)
}
func (g *ginWriterDecorator) CloseNotify() <-chan bool {
	return g.CustomWriter.CloseNotify()
}
func (g *ginWriterDecorator) Flush() {
	g.CustomWriter.Flush()
}
func (g *ginWriterDecorator) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return g.CustomWriter.Hijack()
}
func (g *ginWriterDecorator) Pusher() http.Pusher {
	return g.CustomWriter.Pusher()
}
func (g *ginWriterDecorator) Size() int {
	return g.CustomWriter.Size()
}
func (g *ginWriterDecorator) Status() int {
	return g.CustomWriter.Status()
}
func (g *ginWriterDecorator) WriteHeaderNow() {
	g.CustomWriter.WriteHeaderNow()
}
func (g *ginWriterDecorator) Written() bool {
	return g.CustomWriter.Written()
}
func (g *ginWriterDecorator) WriteString(s string) (int, error) {
	return g.CustomWriter.WriteString(s)
}
