package gin

import (
	"bufio"
	"net"
	"net/http"

	"github.com/darkweak/souin/pkg/middleware"
	"github.com/gin-gonic/gin"
)

var _ (gin.ResponseWriter) = (*ginWriterDecorator)(nil)

type ginWriterDecorator struct {
	CustomWriter *middleware.CustomWriter
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
	return g.CustomWriter.Rw.(gin.ResponseWriter).CloseNotify()
}
func (g *ginWriterDecorator) Flush() {
	g.CustomWriter.Rw.(gin.ResponseWriter).Flush()
}
func (g *ginWriterDecorator) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return g.CustomWriter.Rw.(gin.ResponseWriter).Hijack()
}
func (g *ginWriterDecorator) Pusher() http.Pusher {
	return g.CustomWriter.Rw.(gin.ResponseWriter).Pusher()
}
func (g *ginWriterDecorator) Size() int {
	return g.CustomWriter.Rw.(gin.ResponseWriter).Size()
}
func (g *ginWriterDecorator) Status() int {
	return g.CustomWriter.Rw.(gin.ResponseWriter).Status()
}
func (g *ginWriterDecorator) WriteHeaderNow() {
	g.CustomWriter.Rw.(gin.ResponseWriter).WriteHeaderNow()
}
func (g *ginWriterDecorator) Written() bool {
	return g.CustomWriter.Rw.(gin.ResponseWriter).Written()
}
func (g *ginWriterDecorator) WriteString(s string) (int, error) {
	return g.CustomWriter.Rw.(gin.ResponseWriter).WriteString(s)
}
