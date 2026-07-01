Component({
  properties: {
    items: {
      type: Array,
      value: []
    }
  },
  methods: {
    onEntryTap(event) {
      const index = Number(event.currentTarget.dataset.index)
      this.triggerEvent('entrytap', { index, item: this.data.items[index] })
    }
  }
})
