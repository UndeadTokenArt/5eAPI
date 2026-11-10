(function () {
  var nav = document.querySelector('.site-nav');
  if (!nav) return;
  var toggle = nav.querySelector('.nav-toggle');
  var links = nav.querySelector('.nav-links');
  if (!toggle || !links) return;

  var mobile = window.matchMedia('(max-width: 1100px)');

  function setOpen(open) {
    links.hidden = mobile.matches && !open;
    toggle.setAttribute('aria-expanded', String(mobile.matches && open));
    toggle.setAttribute('aria-label', open && mobile.matches ? 'Close navigation menu' : 'Open navigation menu');
  }

  function updateHeight() {
    document.documentElement.style.setProperty('--site-nav-height', nav.offsetHeight + 'px');
  }

  setOpen(false);
  nav.classList.add('nav-ready');
  updateHeight();
  if (window.ResizeObserver) new ResizeObserver(updateHeight).observe(nav);
  window.addEventListener('resize', updateHeight);
  mobile.addEventListener('change', function () { setOpen(false); updateHeight(); });
  toggle.addEventListener('click', function () {
    setOpen(toggle.getAttribute('aria-expanded') !== 'true');
  });
  document.addEventListener('click', function (event) {
    if (mobile.matches && !nav.contains(event.target)) setOpen(false);
  });
  document.addEventListener('keydown', function (event) {
    if (event.key === 'Escape' && mobile.matches && toggle.getAttribute('aria-expanded') === 'true') {
      setOpen(false);
      toggle.focus();
    }
  });
})();
