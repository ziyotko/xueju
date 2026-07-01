Component({
  properties: {
    tabs: {
      type: Array,
      value: ["推荐", "最新"]
    },
    active: {
      type: String,
      value: "推荐"
    }
  },
  methods: {
    onTabTap(event) {
      this.triggerEvent("change", { active: event.currentTarget.dataset.tab })
    },
    onFilterTap() {
      this.triggerEvent("filter")
    }
  }
})
