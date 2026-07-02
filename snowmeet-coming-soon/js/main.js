(function () {
  var body = document.body;
  var menuToggle = document.querySelector('.menu-toggle');
  var navLinks = document.querySelectorAll('.nav-link');

  if (menuToggle) {
    menuToggle.addEventListener('click', function () {
      var isOpen = body.classList.toggle('nav-open');
      menuToggle.setAttribute('aria-expanded', String(isOpen));
    });
  }

  navLinks.forEach(function (link) {
    link.addEventListener('click', function () {
      body.classList.remove('nav-open');
      if (menuToggle) menuToggle.setAttribute('aria-expanded', 'false');
    });
  });

  var sections = Array.from(document.querySelectorAll('#top, #features, #coming-soon, #contact'));

  function setActiveNav() {
    var scrollTop = window.scrollY + 120;
    var currentId = 'top';

    sections.forEach(function (section) {
      if (section.offsetTop <= scrollTop) {
        currentId = section.id || 'top';
      }
    });

    navLinks.forEach(function (link) {
      var href = link.getAttribute('href') || '';
      link.classList.toggle('active', href === '#' + currentId);
    });
  }

  window.addEventListener('scroll', setActiveNav, { passive: true });
  setActiveNav();
})();
