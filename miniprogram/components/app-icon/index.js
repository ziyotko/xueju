const DEFAULT_COLOR = "#64748B"
const ACTIVE_COLOR = "#4F46E5"
const LOCAL_ICON_EXTENSIONS = {
  snowboard: "svg",
  ski: "svg",
  "ski-resort": "svg",
  helmet: "svg",
  goggles: "svg",
  snowflake: "svg"
}

Component({
  properties: {
    name: {
      type: String,
      value: ""
    },
    size: {
      type: null,
      value: 24
    },
    color: {
      type: String,
      value: DEFAULT_COLOR
    },
    className: {
      type: String,
      value: ""
    },
    active: {
      type: Boolean,
      value: false
    },
    source: {
      type: String,
      value: "tdesign"
    }
  },

  data: {
    iconSize: "24px",
    iconColor: DEFAULT_COLOR,
    localSrc: ""
  },

  observers: {
    "name,size,color,active,source": function () {
      this.updateIcon()
    }
  },

  lifetimes: {
    attached() {
      this.updateIcon()
    }
  },

  methods: {
    updateIcon() {
      const size = Number(this.data.size) || 24
      const name = this.data.name || ""
      const extension = LOCAL_ICON_EXTENSIONS[name] || "png"

      this.setData({
        iconSize: `${size}px`,
        iconColor: this.data.active ? ACTIVE_COLOR : this.data.color || DEFAULT_COLOR,
        localSrc: this.data.source === "local" && name ? `/assets/icons/${name}.${extension}` : ""
      })
    }
  }
})
