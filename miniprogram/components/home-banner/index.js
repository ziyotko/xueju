Component({
  properties: {
    image: {
      type: String,
      value: ""
    },
    title: {
      type: String,
      value: "这个周末\n找个水平差不多的人一起滑"
    },
    titleLines: {
      type: Array,
      value: ["这个周末", "找个水平差不多的人一起滑"]
    },
    subtitle: {
      type: String,
      value: "LET'S SKI TOGETHER"
    },
    buttonText: {
      type: String,
      value: "发布滑雪行程"
    }
  },

  methods: {
    onAction() {
      this.triggerEvent("action")
    }
  }
})
