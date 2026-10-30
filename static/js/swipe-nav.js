// Swipe left on the character sheet to open the campaign screen; swipe right on it to return.
(function () {
  var path = location.pathname;
  var onSheet = path === '/characters';
  var onCampaign = path.indexOf('/campaigns') === 0;
  if (!onSheet && !onCampaign) return;

  var MIN_DISTANCE = 80;
  var start = null;

  function interactive(target) {
    return !!(target.closest && target.closest('input, textarea, select, button, [contenteditable], .dice-log, .suggest-list'));
  }

  function lastCampaign() {
    try { return localStorage.getItem('dnd5e-last-campaign'); } catch (error) { return null; }
  }

  document.addEventListener('touchstart', function (event) {
    if (event.touches.length !== 1 || interactive(event.target)) { start = null; return; }
    start = { x: event.touches[0].clientX, y: event.touches[0].clientY };
  }, { passive: true });

  document.addEventListener('touchend', function (event) {
    if (!start) return;
    var touch = event.changedTouches[0];
    var dx = touch.clientX - start.x;
    var dy = touch.clientY - start.y;
    start = null;
    if (Math.abs(dx) < MIN_DISTANCE || Math.abs(dx) < Math.abs(dy) * 2) return;
    if (onSheet && dx < 0) {
      var id = lastCampaign();
      location.href = id ? '/campaigns/' + encodeURIComponent(id) : '/campaigns';
    } else if (onCampaign && dx > 0) {
      location.href = '/characters';
    }
  }, { passive: true });
})();
