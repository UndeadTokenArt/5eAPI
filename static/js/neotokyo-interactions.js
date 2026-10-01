// neotokyo-interactions.js
document.addEventListener('DOMContentLoaded', () => {
  const sectionsWrapper = document.getElementById('sectionsWrapper');
  const allSectionNodes = document.querySelectorAll('.section-node');
  const originalOrder = Array.from(sectionsWrapper.children);

  // 1. Collapsible TOC
  document.querySelectorAll('.toc-toggle-btn').forEach(btn => {
    btn.addEventListener('click', (e) => {
      const childList = e.target.closest('li').querySelector('.toc-children');
      const isHidden = childList.style.display === 'none';
      childList.style.display = isHidden ? 'block' : 'none';
      e.target.textContent = isHidden ? '▼' : '▶';
    });
  });

  // 2. Search Filtering
  document.getElementById('nodeSearch').addEventListener('input', (e) => {
    const query = e.target.value.toLowerCase();
    allSectionNodes.forEach(node => {
      const textContent = node.textContent.toLowerCase();
      node.style.display = textContent.includes(query) ? 'block' : 'none';
    });
  });

  // 3. Sorting (Applies to top-level sections only)
  document.getElementById('nodeSort').addEventListener('change', (e) => {
    const sortMode = e.target.value;
    if (sortMode === 'default') {
      sectionsWrapper.append(...originalOrder);
      return;
    }

    const sortedNodes = Array.from(sectionsWrapper.children).sort((a, b) => {
      const titleA = (a.getAttribute('data-title') || '').toLowerCase();
      const titleB = (b.getAttribute('data-title') || '').toLowerCase();
      return sortMode === 'asc' ? titleA.localeCompare(titleB) : titleB.localeCompare(titleA);
    });
    
    sectionsWrapper.append(...sortedNodes);
  });

  // 4. Isolate Selected via TOC
  document.querySelectorAll('.toc-link').forEach(link => {
    link.addEventListener('click', (e) => {
      e.preventDefault();
      const targetId = e.target.getAttribute('data-target');
      const targetElement = document.getElementById(targetId);

      if (!targetElement) return;

      // Hide all nodes initially
      allSectionNodes.forEach(node => node.style.display = 'none');

      // Unhide the target and all its nested children
      targetElement.style.display = 'block';
      targetElement.querySelectorAll('.section-node').forEach(child => {
        child.style.display = 'block';
      });

      // Unhide parent chain so the isolated target remains visible
      let parent = targetElement.parentElement;
      while (parent && parent.closest('.section-node')) {
        const parentNode = parent.closest('.section-node');
        parentNode.style.display = 'block';
        parent = parentNode.parentElement;
      }

      targetElement.scrollIntoView({ behavior: 'smooth', block: 'start' });
    });
  });

  // Reset Isolation
  document.getElementById('resetView').addEventListener('click', () => {
    document.getElementById('nodeSearch').value = '';
    allSectionNodes.forEach(node => node.style.display = 'block');
  });
});