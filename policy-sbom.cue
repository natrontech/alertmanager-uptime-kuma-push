// The predicateType field must match this string
predicateType: "https://cyclonedx.org/bom"

predicate: {
  metadata: {
    component: {
      "bom-ref": =~"^pkg:golang/github.com/natrontech/alertmanager-uptime-kuma-push@v[0-9]+.[0-9]+.[0-9]+(-rc.[0-9]+)?\\?type=module#cmd/pusher$"
    }
  }
}
