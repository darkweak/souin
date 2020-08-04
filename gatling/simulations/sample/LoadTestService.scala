package sample

import scala.concurrent.duration._
import io.gatling.core.Predef._
import io.gatling.http.Predef._
import io.gatling.jdbc.Predef._
import java.net.InetAddress

class LoadTestService extends Simulation {
  val httpProtocol = http
    .baseUrl("http://domain.com")
    .acceptHeader("text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
    .acceptLanguageHeader("en-US,en;q=0.5")
    .acceptEncodingHeader("gzip, deflate")
    .userAgentHeader("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_5) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/13.1.1 Safari/605.1.15")

  val scn = scenario("Packager").repeat(1000) {
    exec(
      http("fetch_first_route")
        .get("/")
        .check(status)
    )
  }

  setUp(
    scn.inject(
      nothingFor(4.seconds),
      atOnceUsers(1),
      rampUsers(10) during (10 seconds)
    )
  ).maxDuration(FiniteDuration.apply(5, "minutes")).protocols(httpProtocol)
}
