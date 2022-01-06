package gin

import (
	"bufio"
	"net"
	"net/http"

	"github.com/darkweak/souin/plugins"
	"github.com/gin-gonic/gin"
)

type ginWriterDecorator struct {
	*plugins.CustomWriter
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
