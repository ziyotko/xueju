// Static visual assets only. Business/demo records live in the development
// database seed and are never used as a production fallback.
const unsplashImages = {
  banner: "https://images.unsplash.com/photo-1726688648196-c5d505af321e?auto=format&fit=crop&w=1200&q=80",
  wanlong: "https://images.unsplash.com/photo-1740137660688-3d3f2b5422b6?auto=format&fit=crop&w=900&q=80",
  nanshan: "https://images.unsplash.com/photo-1678973751386-66d7bd0f5abf?auto=format&fit=crop&w=900&q=80",
  yunding: "https://images.unsplash.com/photo-1707290796500-95016769aa11?auto=format&fit=crop&w=900&q=80",
  taiwu: "https://images.unsplash.com/photo-1558733467-11cef06eb6d8?auto=format&fit=crop&w=900&q=80",
  profile: "https://images.unsplash.com/photo-1726688648196-c5d505af321e?auto=format&fit=crop&w=1200&q=80"
}

module.exports = {
  unsplashImages,
  heroImage: unsplashImages.banner,
  profileCover: unsplashImages.profile
}
