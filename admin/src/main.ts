import { createApp } from "vue"
import {
  ElAlert,
  ElAside,
  ElButton,
  ElConfigProvider,
  ElContainer,
  ElDescriptions,
  ElDescriptionsItem,
  ElDialog,
  ElDrawer,
  ElEmpty,
  ElForm,
  ElFormItem,
  ElHeader,
  ElIcon,
  ElImage,
  ElInput,
  ElInputNumber,
  ElMain,
  ElMenu,
  ElMenuItem,
  ElMenuItemGroup,
  ElOption,
  ElPagination,
  ElSelect,
  ElTabPane,
  ElTable,
  ElTableColumn,
  ElTabs,
  ElTag,
  ElTimeline,
  ElTimelineItem,
  ElUpload
} from "element-plus"
import "element-plus/dist/index.css"

import App from "./App.vue"
import router from "./router"
import "./styles/base.css"

const app = createApp(App)
const components = [
  ElAlert, ElAside, ElButton, ElConfigProvider, ElContainer, ElDescriptions,
  ElDescriptionsItem, ElDialog, ElDrawer, ElEmpty, ElForm, ElFormItem, ElHeader,
  ElIcon, ElImage, ElInput, ElInputNumber, ElMain, ElMenu, ElMenuItem,
  ElMenuItemGroup, ElOption, ElPagination, ElSelect, ElTabPane, ElTable,
  ElTableColumn, ElTabs, ElTag, ElTimeline, ElTimelineItem, ElUpload
]

components.forEach((component) => app.component(component.name!, component))
app.use(router).mount("#app")
