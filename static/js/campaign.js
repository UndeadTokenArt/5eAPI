// Campaign screen: list/create, join by link, and the live websocket-synced tracker.
(function () {
  var root = document.getElementById('campaign-root');
  if (!root) return;
  var mode = root.dataset.mode;
  var statusBox = document.getElementById('campaign-status');
  var CONDITIONS = ['Optic Glitch', 'Subliminal Override', 'Acoustic Scramble', 'Cyber-Strain', 'Threat-Lock', 'OS Buffer Overflow',
    'Optical Camouflage', 'Neural Freeze', 'Solid-State Stasis', 'Bio-Toxed', 'Grounded', 'Mag-Bound', 'Synapse Short', 'Flatlined'];
  var DICE = [4, 6, 8, 10, 12, 20];

  function setStatus(message, isError) {
    statusBox.textContent = message || '';
    statusBox.classList.toggle('error', !!isError);
  }

  async function api(path, options) {
    var init = Object.assign({ credentials: 'same-origin' }, options || {});
    if (init.body !== undefined) {
      init.headers = { 'Content-Type': 'application/json' };
      init.body = JSON.stringify(init.body);
    }
    var response = await fetch(path, init);
    var data = response.status === 204 ? null : await response.json();
    if (!response.ok) throw new Error(data && data.error ? data.error : 'Request failed.');
    return data;
  }

  function el(tag, className, text) {
    var node = document.createElement(tag);
    if (className) node.className = className;
    if (text != null) node.textContent = text;
    return node;
  }

  function show(id) { document.getElementById(id).hidden = false; }

  // ---- list ----
  async function initList() {
    show('campaign-list');
    var items = document.getElementById('campaign-items');
    async function refresh() {
      var list = await api('/api/campaigns');
      items.replaceChildren();
      if (!list.length) items.appendChild(el('li', 'muted', 'You are not in any campaigns yet.'));
      list.forEach(function (campaign) {
        var li = el('li');
        var link = el('a');
        link.href = '/campaigns/' + encodeURIComponent(campaign.id);
        link.append(el('span', '', campaign.name), el('small', '', (campaign.is_owner ? 'Creator · ' : '') + campaign.members + ' character(s)'));
        li.appendChild(link);
        items.appendChild(li);
      });
    }
    document.getElementById('campaign-create').addEventListener('submit', async function (event) {
      event.preventDefault();
      try {
        var created = await api('/api/campaigns', { method: 'POST', body: { name: document.getElementById('campaign-name').value } });
        location.href = '/campaigns/' + encodeURIComponent(created.id);
      } catch (error) { setStatus(error.message, true); }
    });
    try { await refresh(); } catch (error) { setStatus(error.message, true); }
  }

  // ---- join ----
  async function initJoin() {
    show('campaign-join');
    var token = encodeURIComponent(root.dataset.joinToken);
    try {
      var info = await api('/api/campaign-join/' + token);
      document.getElementById('campaign-title').textContent = 'Join ' + info.name;
      var characters = await api('/api/characters');
      var box = document.getElementById('join-characters');
      document.getElementById('join-intro').textContent = characters.length
        ? 'Choose which characters to add. Their name, species, class, HP, AC, initiative, conditions, cyber strain, portrait and rolls will be visible to everyone in this campaign.'
        : 'You have no saved characters yet. Create one on the Characters page, then reopen this link.';
      characters.forEach(function (character) {
        var label = el('label');
        var box1 = document.createElement('input');
        box1.type = 'checkbox';
        box1.value = character.id;
        label.append(box1, document.createTextNode(character.name + (character.class ? ' · ' + character.class : '')));
        box.appendChild(label);
      });
      document.getElementById('join-submit').addEventListener('click', async function () {
        var ids = Array.prototype.map.call(box.querySelectorAll('input:checked'), function (input) { return input.value; });
        if (!ids.length) { setStatus('Choose at least one character.', true); return; }
        try {
          var joined = await api('/api/campaign-join/' + token, { method: 'POST', body: { character_ids: ids } });
          location.href = '/campaigns/' + encodeURIComponent(joined.id);
        } catch (error) { setStatus(error.message, true); }
      });
    } catch (error) { setStatus(error.message, true); document.getElementById('join-submit').hidden = true; }
  }

  // ---- live tracker ----
  function initView() {
    show('campaign-view');
    var id = root.dataset.campaignId;
    try { localStorage.setItem('dnd5e-last-campaign', id); } catch (error) { /* storage unavailable */ }
    var party = document.getElementById('party');
    var connection = document.getElementById('connection');
    var latest = null;
    var pending = null;
    var socket = null;
    var attempt = 0;
    var reconnectTimer = null;

    function setConnection(text, cls) {
      connection.textContent = text;
      connection.className = 'connection ' + (cls || '');
    }

    function memberURL(member) {
      return '/api/campaigns/' + encodeURIComponent(id) + (member.kind === 'npc' ? '/npcs/' : '/members/') + encodeURIComponent(member.character_id);
    }

    async function patch(member, body) {
      try {
        await api(memberURL(member), { method: 'PATCH', body: body });
        setStatus('');
      } catch (error) { setStatus(error.message, true); render(latest); }
    }

    function stat(label) {
      var wrap = el('div', 'stat');
      wrap.appendChild(el('label', '', label));
      return wrap;
    }

    function numberInput(value, min, max, onCommit) {
      var input = document.createElement('input');
      input.type = 'number';
      input.min = min;
      input.max = max;
      input.value = value == null ? '' : value;
      input.addEventListener('change', function () { onCommit(input.value === '' ? null : Number(input.value)); });
      return input;
    }

    var expanded = {};

    function renderCompact(member) {
      var li = el('li', 'member compact');
      if (member.max_hp > 0 && member.current_hp === 0) li.classList.add('is-down');
      li.appendChild(el('h3', 'member-name', member.name));
      li.appendChild(el('span', 'compact-hp', 'HP ' + member.current_hp + (member.max_hp > 0 ? ' / ' + member.max_hp : '')));
      var open = el('button', '', 'Expand');
      open.type = 'button';
      open.setAttribute('aria-label', 'Expand ' + member.name);
      open.addEventListener('click', function () { expanded[member.character_id] = true; render(latest); });
      li.appendChild(open);
      return li;
    }

    function renderMember(member) {
      var turnOn = !!(latest && latest.active_id);
      var isActive = turnOn && member.character_id === latest.active_id;
      if (turnOn && !isActive && !expanded[member.character_id]) return renderCompact(member);
      var li = el('li', 'member');
      if (isActive) li.classList.add('is-active');
      if (member.max_hp > 0 && member.current_hp === 0) li.classList.add('is-down');
      if (member.portrait) {
        var img = el('img', 'member-portrait');
        img.src = member.portrait;
        img.alt = '';
        img.referrerPolicy = 'no-referrer';
        img.loading = 'lazy';
        li.appendChild(img);
      } else {
        li.appendChild(el('div', 'member-portrait empty', (member.name || '?').charAt(0).toUpperCase()));
      }
      var body = el('div');
      var head = el('div', 'member-head');
      head.appendChild(el('h3', 'member-name', member.name));
      head.appendChild(el('span', 'member-sub', member.kind === 'npc' ? 'Non-player entity' : [member.species, member['class']].filter(Boolean).join(' · ') + ' — ' + member.player));
      if (turnOn && !isActive) {
        var fold = el('button', 'fold', 'Collapse');
        fold.type = 'button';
        fold.addEventListener('click', function () { delete expanded[member.character_id]; render(latest); });
        head.appendChild(fold);
      }
      body.appendChild(head);

      var stats = el('div', 'member-stats');
      var hp = stat('HP');
      if (member.can_edit) {
        var stepper = el('div', 'stepper');
        var minus = el('button', '', '−');
        var plus = el('button', '', '+');
        minus.type = plus.type = 'button';
        minus.setAttribute('aria-label', 'Lower HP');
        plus.setAttribute('aria-label', 'Raise HP');
        var hpInput = numberInput(member.current_hp, 0, 9999, function (value) { if (value != null) patch(member, { current_hp: value }); });
        minus.addEventListener('click', function () { patch(member, { current_hp: Math.max(0, member.current_hp - 1) }); });
        plus.addEventListener('click', function () { patch(member, { current_hp: Math.min(9999, member.current_hp + 1) }); });
        stepper.append(minus, hpInput, plus);
        hp.appendChild(stepper);
      } else {
        hp.appendChild(el('output', '', String(member.current_hp)));
      }
      if (member.max_hp > 0) {
        hp.firstChild.textContent = 'HP / ' + member.max_hp;
        var bar = el('div', 'hp-bar');
        var fill = el('span');
        var ratio = Math.max(0, Math.min(1, member.current_hp / member.max_hp));
        fill.style.width = Math.round(ratio * 100) + '%';
        bar.appendChild(fill);
        if (ratio <= 0.25) bar.classList.add('low');
        hp.appendChild(bar);
      }
      stats.appendChild(hp);

      if (member.kind !== 'npc') {
        var ac = stat('AC');
        ac.appendChild(el('output', '', String(member.armor_class)));
        stats.appendChild(ac);
      }

      var init = stat('Initiative');
      if (member.can_edit) {
        init.appendChild(numberInput(member.initiative, -20, 50, function (value) {
          patch(member, value == null ? { clear_initiative: true } : { initiative: value });
        }));
      } else {
        init.appendChild(el('output', '', member.initiative == null ? '—' : String(member.initiative)));
      }
      stats.appendChild(init);

      if (member.kind !== 'npc') {
        var strain = stat('Cyber strain');
        if (member.can_edit) {
          strain.appendChild(numberInput(member.cyber_strain, 0, 20, function (value) { patch(member, { cyber_strain: value == null ? 0 : value }); }));
        } else {
          strain.appendChild(el('output', '', String(member.cyber_strain)));
        }
        stats.appendChild(strain);
      }
      body.appendChild(stats);

      var conditions = el('div', 'conditions');
      (member.conditions || []).forEach(function (condition) {
        var chip = el('span', 'chip', condition);
        if (member.can_edit) {
          var remove = el('button', '', '×');
          remove.type = 'button';
          remove.setAttribute('aria-label', 'Remove ' + condition);
          remove.addEventListener('click', function () {
            patch(member, { conditions: member.conditions.filter(function (item) { return item !== condition; }) });
          });
          chip.appendChild(remove);
        }
        conditions.appendChild(chip);
      });
      body.appendChild(conditions);

      if (member.kind === 'npc' && member.can_roll && (member.attacks || []).length) body.appendChild(renderAttacks(member));
      if (member.can_edit) {
        var add = el('div', 'condition-add');
        var input = document.createElement('select');
        input.setAttribute('aria-label', 'Add condition to ' + member.name);
        input.appendChild(new Option('Add condition…', ''));
        CONDITIONS.filter(function (name) { return (member.conditions || []).indexOf(name) < 0; }).forEach(function (name) { input.appendChild(new Option(name, name)); });
        input.addEventListener('change', function () {
          if (input.value) patch(member, { conditions: (member.conditions || []).concat(input.value) });
        });
        add.appendChild(input);
        body.appendChild(add);

        var actions = el('div', 'member-actions');
        var leave = el('button', 'danger', member.kind === 'npc' ? 'Remove' : 'Remove from campaign');
        leave.type = 'button';
        leave.addEventListener('click', async function () {
          if (!window.confirm('Remove ' + member.name + ' from this campaign?')) return;
          try { await api(memberURL(member), { method: 'DELETE' }); }
          catch (error) { setStatus(error.message, true); }
        });
        actions.appendChild(leave);
        body.appendChild(actions);
      }
      li.appendChild(body);
      return li;
    }

    // ---- non-player entity attacks ----
    // Shift-click rolls with advantage, Alt-click with disadvantage. A natural 20 on the to-hit
    // roll doubles the dice on that attack's next damage roll.
    var critPending = {};

    function signedNumber(n) { return (n < 0 ? '- ' : '+ ') + Math.abs(n); }

    function parseDice(text) {
      var m = /^\s*(\d+)d(\d+)\s*(?:([+-])\s*(\d+))?\s*$/i.exec(text);
      if (!m || Number(m[1]) < 1 || Number(m[1]) > 50 || Number(m[2]) < 2) return null;
      return { count: Number(m[1]), sides: Number(m[2]), bonus: m[3] ? (m[3] === '-' ? -1 : 1) * Number(m[4]) : 0 };
    }

    async function postRoll(member, label, detail, total, note) {
      try {
        await api('/api/campaigns/' + encodeURIComponent(id) + '/rolls', { method: 'POST', body: { character_id: member.character_id, label: label, detail: detail, total: total, note: note || '' } });
        setStatus('');
      } catch (error) { setStatus(error.message, true); }
    }

    function rollToHit(member, attack, event) {
      var first = rollDie(20);
      var mode = event.shiftKey ? 'advantage' : event.altKey ? 'disadvantage' : 'normal';
      var second = mode === 'normal' ? null : rollDie(20);
      var kept = second === null ? first : mode === 'advantage' ? Math.max(first, second) : Math.min(first, second);
      var dice = second === null ? 'd20 [' + first + ']' : 'd20 [' + first + ', ' + second + '] keep ' + kept + ' (' + mode + ')';
      var total = kept + attack.attack_bonus;
      critPending[member.character_id + '|' + attack.name] = kept === 20;
      postRoll(member, attack.name + ' attack', dice + ' ' + signedNumber(attack.attack_bonus) + ' = ' + total, total, kept === 20 ? 'Natural 20' : kept === 1 ? 'Natural 1' : '');
    }

    function rollAttackDamage(member, attack) {
      var key = member.character_id + '|' + attack.name;
      var crit = Boolean(critPending[key]);
      delete critPending[key];
      var total = 0;
      var parts = [];
      for (var i = 0; i < attack.damage.length; i++) {
        var spec = parseDice(attack.damage[i].dice);
        if (!spec) { setStatus('Cannot roll "' + attack.damage[i].dice + '" automatically.', true); return; }
        var count = spec.count * (crit ? 2 : 1);
        var rolls = [];
        var sum = spec.bonus;
        for (var n = 0; n < count; n++) { var value = rollDie(spec.sides); rolls.push(value); sum += value; }
        total += sum;
        parts.push(count + 'd' + spec.sides + ' [' + rolls.join(', ') + ']' + (spec.bonus ? ' ' + signedNumber(spec.bonus) : '') + ' ' + (attack.damage[i].type || '').toLowerCase());
      }
      postRoll(member, attack.name + ' damage', parts.join(' + ').replace(/\s+/g, ' ').trim() + ' = ' + total, total, crit ? 'Critical dice doubled' : '');
    }

    function renderAttacks(member) {
      var box = el('div', 'npc-attacks');
      member.attacks.forEach(function (attack) {
        var row = el('div', 'npc-attack');
        row.appendChild(el('span', 'npc-attack-name', attack.name));
        var tip = (attack.description || attack.name) + '\n(Shift-click to hit: advantage. Alt-click: disadvantage.)';
        var hit = el('button', '', 'Hit ' + (attack.attack_bonus < 0 ? '' : '+') + attack.attack_bonus);
        hit.type = 'button';
        hit.title = tip;
        hit.setAttribute('aria-label', attack.name + ' attack roll, ' + (attack.attack_bonus < 0 ? '' : '+') + attack.attack_bonus + '. ' + (attack.description || ''));
        hit.addEventListener('click', function (event) { rollToHit(member, attack, event); });
        var dmg = el('button', '', 'Dmg ' + attack.damage.map(function (d) { return d.dice; }).join(' + '));
        dmg.type = 'button';
        dmg.title = (attack.description || attack.name) + '\nDamage: ' + attack.damage.map(function (d) { return d.dice + (d.type ? ' ' + d.type.toLowerCase() : ''); }).join(' + ') + '\n(After a natural 20 the dice are doubled.)';
        dmg.setAttribute('aria-label', attack.name + ' damage roll. ' + (attack.description || ''));
        dmg.addEventListener('click', function () { rollAttackDamage(member, attack); });
        row.append(hit, dmg);
        box.appendChild(row);
      });
      return box;
    }

    function renderRolls(rolls) {
      var log = document.getElementById('dice-log');
      log.replaceChildren();
      document.getElementById('dice-empty').hidden = rolls.length > 0;
      rolls.forEach(function (roll) {
        var li = el('li');
        if (roll.total != null) li.appendChild(el('span', 'roll-total', String(roll.total)));
        li.appendChild(el('span', 'roll-who', roll.player_name));
        li.appendChild(document.createTextNode(' · ' + roll.label + (roll.note ? ' (' + roll.note + ')' : '')));
        if (roll.detail) li.appendChild(el('small', '', roll.detail));
        log.appendChild(li);
      });
    }

    function renderTurn(state) {
      var box = document.getElementById('turn-box');
      var next = document.getElementById('turn-next');
      var end = document.getElementById('turn-end');
      var current = state.members.filter(function (m) { return m.character_id === state.active_id; })[0];
      var mine = !!current && current.kind === 'pc' && current.can_roll;
      var canAdvance = state.is_owner || mine;
      var hasInit = state.members.some(function (m) { return m.initiative != null; });
      box.hidden = !(state.round > 0 || (state.is_owner && hasInit));
      document.getElementById('turn-status').textContent = state.round > 0
        ? 'Round ' + state.round + (current ? ' · ' + current.name + '\u2019s turn' : '')
        : 'Combat has not started.';
      next.textContent = state.round > 0 ? 'Next turn' : 'Start combat';
      next.hidden = !canAdvance;
      end.hidden = !(state.is_owner && state.round > 0);
    }

    function render(state) {
      if (!state) return;
      latest = state;
      // Do not rebuild while someone is typing; apply the newest state when focus leaves.
      var active = document.activeElement;
      if (active && party.contains(active) && active.tagName === 'INPUT') { pending = state; return; }
      pending = null;
      document.getElementById('campaign-title').textContent = state.name;
      document.title = state.name + ' | Campaign';
      party.replaceChildren();
      state.members.forEach(function (member) { party.appendChild(renderMember(member)); });
      document.getElementById('party-empty').hidden = state.members.length > 0;
      renderRolls(state.rolls);
      renderTurn(state);
      document.getElementById('expiry').textContent = 'Shared info is refreshed from character sheets after 48 hours of inactivity (next: ' + new Date(state.expires_at).toLocaleString() + ').';
      document.getElementById('npc-form').hidden = !state.is_owner;
      updateRollers(state);
      if (state.is_owner) {
        loadDropboxCatalog();
        document.getElementById('share-box').hidden = false;
        document.getElementById('share-link').value = location.origin + state.join_path;
        document.getElementById('campaign-leave').hidden = false;
        var box = document.getElementById('gm-dropbox');
        var itemList = document.getElementById('dropbox-items');
        var recipient = document.getElementById('dropbox-recipient');
        var previousRecipient = recipient.value;
        box.hidden = false;
        itemList.replaceChildren();
        (state.dropbox_items || []).forEach(function (item) {
          itemList.appendChild(el('li', '', item.name + ' × ' + item.quantity));
        });
        document.getElementById('dropbox-empty').hidden = (state.dropbox_items || []).length > 0 || state.dropbox_yen > 0;
        document.getElementById('dropbox-yen').textContent = String(state.dropbox_yen || 0);
        document.getElementById('dropbox-clear').disabled = !(state.dropbox_items || []).length && !state.dropbox_yen;
        recipient.replaceChildren();
        state.members.filter(function (member) { return member.kind === 'pc' && member.send_alias; }).forEach(function (member) {
          recipient.add(new Option(member.name, member.send_alias));
        });
        if (previousRecipient && Array.prototype.some.call(recipient.options, function (o) { return o.value === previousRecipient; })) recipient.value = previousRecipient;
        document.getElementById('dropbox-send').querySelector('button').disabled = recipient.options.length === 0;
      }
    }

    var dropboxItemName = document.getElementById('dropbox-item-name');
    var dropboxSuggestions = document.getElementById('dropbox-item-suggestions');
    var dropboxCatalog = [];
    var activeDropboxSuggestion = -1;
    var currentDropboxMatches = [];
    var dropboxCatalogRequested = false;

    function loadDropboxCatalog() {
      if (dropboxCatalogRequested) return;
      dropboxCatalogRequested = true;
      api('/api/character-options').then(function (options) {
        dropboxCatalog = (options.equipment || []).concat(options.weapons || []);
        if (document.activeElement === dropboxItemName) renderDropboxSuggestions();
      }).catch(function (error) {
        setStatus('Could not load item suggestions: ' + error.message, true);
      });
    }

    function closeDropboxSuggestions() {
      dropboxSuggestions.hidden = true;
      dropboxItemName.setAttribute('aria-expanded', 'false');
      activeDropboxSuggestion = -1;
    }

    function pickDropboxSuggestion(item) {
      dropboxItemName.value = item.name;
      closeDropboxSuggestions();
    }

    function matchDropboxItems(query) {
      var normalized = query.toLowerCase();
      var terms = normalized.split(/\s+/).filter(Boolean);
      if (!terms.length) return dropboxCatalog.slice(0, 12);
      return dropboxCatalog.map(function (item) {
        var name = item.name.toLowerCase();
        var searchable = [name, item.group || '', item.detail || ''].join(' ').toLowerCase();
        if (!terms.every(function (term) { return searchable.indexOf(term) >= 0; })) return null;
        var score = name === normalized ? 0 : name.indexOf(normalized) === 0 ? 1 :
          name.split(/[^a-z0-9]+/).some(function (word) { return word.indexOf(normalized) === 0; }) ? 2 :
            name.indexOf(normalized) >= 0 ? 3 : 4;
        return { item: item, score: score };
      }).filter(Boolean).sort(function (a, b) {
        return a.score - b.score || a.item.name.localeCompare(b.item.name);
      }).slice(0, 10).map(function (match) { return match.item; });
    }

    function renderDropboxSuggestions() {
      var browsing = !dropboxItemName.value.trim();
      var matches = matchDropboxItems(dropboxItemName.value.trim());
      currentDropboxMatches = matches;
      dropboxSuggestions.replaceChildren();
      var optionElements = [];
      var lastGroup = null;
      matches.forEach(function (item) {
        if (browsing && item.group && item.group !== lastGroup) {
          var heading = el('li', 'group-heading', item.group);
          heading.setAttribute('role', 'presentation');
          heading.addEventListener('mousedown', function (event) { event.preventDefault(); });
          dropboxSuggestions.appendChild(heading);
          lastGroup = item.group;
        }
        var option = el('li', '', item.name);
        option.setAttribute('role', 'option');
        var detail = [item.group, item.detail].filter(Boolean).join(' · ');
        if (detail) option.appendChild(el('small', '', detail));
        option.addEventListener('mousedown', function (event) {
          event.preventDefault();
          pickDropboxSuggestion(item);
        });
        dropboxSuggestions.appendChild(option);
        optionElements.push(option);
      });
      activeDropboxSuggestion = matches.length && !browsing ? 0 : -1;
      optionElements.forEach(function (option, index) {
        option.setAttribute('aria-selected', String(index === activeDropboxSuggestion));
      });
      dropboxSuggestions.hidden = !matches.length;
      dropboxItemName.setAttribute('aria-expanded', String(matches.length > 0));
    }

    dropboxItemName.addEventListener('input', renderDropboxSuggestions);
    dropboxItemName.addEventListener('focus', renderDropboxSuggestions);
    dropboxItemName.addEventListener('blur', closeDropboxSuggestions);
    dropboxItemName.addEventListener('keydown', function (event) {
      var options = dropboxSuggestions.querySelectorAll('[role="option"]');
      if (dropboxSuggestions.hidden || !options.length) return;
      if (event.key === 'ArrowDown') {
        event.preventDefault();
        activeDropboxSuggestion = (activeDropboxSuggestion + 1) % options.length;
      } else if (event.key === 'ArrowUp') {
        event.preventDefault();
        activeDropboxSuggestion = (activeDropboxSuggestion - 1 + options.length) % options.length;
      } else if (event.key === 'Enter' && activeDropboxSuggestion >= 0) {
        event.preventDefault();
        pickDropboxSuggestion(currentDropboxMatches[activeDropboxSuggestion]);
        return;
      } else if (event.key === 'Escape') {
        closeDropboxSuggestions();
        return;
      } else {
        return;
      }
      options.forEach(function (option, index) {
        option.setAttribute('aria-selected', String(index === activeDropboxSuggestion));
      });
      options[activeDropboxSuggestion].scrollIntoView({ block: 'nearest' });
    });

    document.getElementById('dropbox-add').addEventListener('submit', async function (event) {
      event.preventDefault();
      var name = dropboxItemName;
      var quantity = Number(document.getElementById('dropbox-item-quantity').value);
      var yen = Number(document.getElementById('dropbox-add-yen').value) || 0;
      var items = name.value.trim() ? [{ name: name.value, quantity: quantity }] : [];
      try {
        var updated = await api('/api/campaigns/' + encodeURIComponent(id) + '/dropbox', {
          method: 'POST', body: { items: items, yen: yen }
        });
        render(updated);
        name.value = '';
        document.getElementById('dropbox-item-quantity').value = '1';
        document.getElementById('dropbox-add-yen').value = '0';
        setStatus('');
      } catch (error) { setStatus(error.message, true); }
    });

    document.getElementById('loot-generator').addEventListener('submit', async function (event) {
      event.preventDefault();
      try {
        var updated = await api('/api/campaigns/' + encodeURIComponent(id) + '/loot', {
          method: 'POST', body: { cr: Number(document.getElementById('loot-cr').value) }
        });
        render(updated);
        setStatus('Generated loot for the dropbox.');
      } catch (error) { setStatus(error.message, true); }
    });

    document.getElementById('dropbox-send').addEventListener('submit', async function (event) {
      event.preventDefault();
      var alias = document.getElementById('dropbox-recipient').value;
      if (!alias) return;
      try {
        var updated = await api('/api/campaigns/' + encodeURIComponent(id) + '/senditem/' + encodeURIComponent(alias), { method: 'POST', body: {} });
        render(updated);
        setStatus('Delivery sent. The recipient can accept or deny it on their character sheet.');
      } catch (error) { setStatus(error.message, true); }
    });

    document.getElementById('dropbox-clear').addEventListener('click', async function () {
      if (!window.confirm('Empty the campaign dropbox? Its items and Yen will be permanently discarded.')) return;
      try {
        var updated = await api('/api/campaigns/' + encodeURIComponent(id) + '/dropbox', { method: 'DELETE' });
        render(updated);
        setStatus('Campaign dropbox emptied.');
      } catch (error) { setStatus(error.message, true); }
    });

    party.addEventListener('focusout', function () {
      window.setTimeout(function () {
        var active = document.activeElement;
        if (pending && !(active && party.contains(active) && active.tagName === 'INPUT')) render(pending);
      }, 0);
    });

    function connect() {
      window.clearTimeout(reconnectTimer);
      var scheme = location.protocol === 'https:' ? 'wss://' : 'ws://';
      socket = new WebSocket(scheme + location.host + '/ws/campaigns/' + encodeURIComponent(id));
      socket.onopen = function () { attempt = 0; setConnection('Live', 'live'); };
      socket.onmessage = function (event) {
        var message;
        try { message = JSON.parse(event.data); } catch (error) { return; }
        if (message.type === 'state' && message.campaign) render(message.campaign);
      };
      socket.onclose = function () {
        setConnection('Reconnecting…', 'down');
        attempt += 1;
        var delay = Math.min(15000, 500 * Math.pow(2, attempt)) * (0.75 + Math.random() * 0.5);
        reconnectTimer = window.setTimeout(resync, delay);
      };
      socket.onerror = function () { socket.close(); };
    }

    // Reconnect only if the campaign is still reachable; otherwise explain why.
    async function resync() {
      try {
        render(await api('/api/campaigns/' + encodeURIComponent(id)));
        connect();
      } catch (error) {
        if (/not found/i.test(error.message)) { setConnection('Unavailable', 'down'); setStatus('This campaign no longer exists or you were removed from it.', true); return; }
        reconnectTimer = window.setTimeout(resync, 5000);
      }
    }

    // Wake-ups from sleeping phones are common mid-game: refresh immediately when the tab returns.
    document.addEventListener('visibilitychange', function () {
      if (document.visibilityState !== 'visible') return;
      if (!socket || socket.readyState !== WebSocket.OPEN) { window.clearTimeout(reconnectTimer); resync(); }
      else socket.send(JSON.stringify({ type: 'sync' }));
    });

    // ---- non-player entities (creator only) ----
    // Searchable catalog; typing a name that is not in it still adds a custom entity.
    var catalog = [];
    var npcName = document.getElementById('npc-name');
    var npcHp = document.getElementById('npc-hp');
    var suggestions = document.getElementById('npc-suggestions');
    var activeSuggestion = -1;
    api('/api/non-player-entities').then(function (list) { catalog = list || []; }).catch(function () {});

    function matchCatalog(query) {
      var terms = query.toLowerCase().split(/\s+/).filter(Boolean);
      if (!terms.length) return [];
      return catalog.map(function (entry) {
        var name = entry.name.toLowerCase();
        var hay = name + ' ' + (entry.summary || '').toLowerCase();
        if (!terms.every(function (t) { return hay.indexOf(t) >= 0; })) return null;
        var score = name === query.toLowerCase() ? 0 : name.indexOf(terms[0]) === 0 ? 1 : name.indexOf(terms[0]) > 0 ? 2 : 3;
        return { entry: entry, score: score };
      }).filter(Boolean).sort(function (a, b) { return a.score - b.score || a.entry.name.localeCompare(b.entry.name); })
        .slice(0, 8).map(function (m) { return m.entry; });
    }
    function closeSuggestions() { suggestions.hidden = true; npcName.setAttribute('aria-expanded', 'false'); activeSuggestion = -1; }
    var pickedEntity = '';
    function pickSuggestion(entry) {
      npcName.value = entry.name;
      pickedEntity = entry.name;
      npcHp.value = entry.hit_points || '';
      closeSuggestions();
    }
    function highlight() {
      Array.prototype.forEach.call(suggestions.children, function (li, i) { li.setAttribute('aria-selected', i === activeSuggestion ? 'true' : 'false'); });
    }
    npcName.addEventListener('input', function () {
      var matches = matchCatalog(npcName.value.trim());
      suggestions.replaceChildren();
      matches.forEach(function (entry) {
        var li = el('li', '', entry.name);
        li.setAttribute('role', 'option');
        var detail = [entry.summary, 'CR ' + entry.challenge_rating, entry.hit_points ? entry.hit_points + ' HP' : '', entry.armor_class ? 'AC ' + entry.armor_class : ''].filter(Boolean).join(' · ');
        li.append(el('small', '', detail));
        li.addEventListener('mousedown', function (event) { event.preventDefault(); pickSuggestion(entry); });
        suggestions.append(li);
      });
      suggestions.hidden = !matches.length;
      npcName.setAttribute('aria-expanded', String(matches.length > 0));
      activeSuggestion = -1;
    });
    npcName.addEventListener('keydown', function (event) {
      var count = suggestions.children.length;
      if (suggestions.hidden || !count) return;
      if (event.key === 'ArrowDown') { event.preventDefault(); activeSuggestion = (activeSuggestion + 1) % count; highlight(); }
      else if (event.key === 'ArrowUp') { event.preventDefault(); activeSuggestion = (activeSuggestion - 1 + count) % count; highlight(); }
      else if (event.key === 'Enter' && activeSuggestion >= 0) { event.preventDefault(); pickSuggestion(matchCatalog(npcName.value.trim())[activeSuggestion]); }
      else if (event.key === 'Escape') closeSuggestions();
    });
    npcName.addEventListener('blur', closeSuggestions);

    document.getElementById('npc-form').addEventListener('submit', async function (event) {
      event.preventDefault();
      var name = document.getElementById('npc-name');
      var hp = document.getElementById('npc-hp');
      var body = { name: name.value, entity: pickedEntity || name.value };
      if (hp.value !== '') body.max_hp = Number(hp.value);
      try {
        await api('/api/campaigns/' + encodeURIComponent(id) + '/npcs', { method: 'POST', body: body });
        name.value = '';
        hp.value = '';
        pickedEntity = '';
        setStatus('');
      } catch (error) { setStatus(error.message, true); }
    });

    // ---- dice hand ----
    var hand = {};
    var rollAs = document.getElementById('roll-as');
    var handBox = document.getElementById('dice-hand');
    var rollButton = document.getElementById('dice-roll');

    function rollDie(sides) {
      var buffer = new Uint32Array(1);
      var limit = Math.floor(4294967296 / sides) * sides;
      do { crypto.getRandomValues(buffer); } while (buffer[0] >= limit);
      return buffer[0] % sides + 1;
    }

    function dieIcon(sides) {
      var ns = 'http://www.w3.org/2000/svg';
      var svg = document.createElementNS(ns, 'svg');
      svg.setAttribute('viewBox', '0 0 40 40');
      svg.setAttribute('aria-hidden', 'true');
      var shape = document.createElementNS(ns, 'polygon');
      var points = { 4: '20,4 37,34 3,34', 6: '6,6 34,6 34,34 6,34', 8: '20,3 36,20 20,37 4,20', 10: '20,3 36,15 30,35 10,35 4,15', 12: '20,3 35,14 30,33 10,33 5,14', 20: '20,2 36,11 36,29 20,38 4,29 4,11' };
      shape.setAttribute('points', points[sides]);
      var text = document.createElementNS(ns, 'text');
      text.setAttribute('x', '20');
      text.setAttribute('y', '25');
      text.setAttribute('text-anchor', 'middle');
      text.textContent = String(sides);
      svg.append(shape, text);
      return svg;
    }

    function renderHand() {
      var parts = DICE.filter(function (sides) { return hand[sides]; }).map(function (sides) { return hand[sides] + 'd' + sides; });
      handBox.textContent = parts.length ? 'Hand: ' + parts.join(' + ') : 'Click dice to add them to your hand.';
      rollButton.disabled = !parts.length || !rollAs.value;
      document.querySelectorAll('[data-die]').forEach(function (button) {
        var count = hand[button.dataset.die] || 0;
        button.querySelector('.die-count').textContent = count ? String(count) : '';
      });
    }

    var diceTray = document.getElementById('dice-tray');
    DICE.forEach(function (sides) {
      var button = document.createElement('button');
      button.type = 'button';
      button.className = 'die';
      button.dataset.die = sides;
      button.setAttribute('aria-label', 'Add a d' + sides + ' to your hand');
      button.append(dieIcon(sides), el('span', 'die-count'));
      button.addEventListener('click', function () {
        var total = DICE.reduce(function (sum, d) { return sum + (hand[d] || 0); }, 0);
        if (total >= 20) { setStatus('A hand holds at most 20 dice.', true); return; }
        hand[sides] = (hand[sides] || 0) + 1;
        renderHand();
      });
      diceTray.appendChild(button);
    });
    async function turnRequest(method) {
      try { await api('/api/campaigns/' + encodeURIComponent(id) + '/turn', { method: method }); setStatus(''); }
      catch (error) { setStatus(error.message, true); }
    }
    document.getElementById('turn-next').addEventListener('click', function () { turnRequest('POST'); });
    document.getElementById('turn-end').addEventListener('click', function () {
      if (window.confirm('End combat and reset the round counter?')) turnRequest('DELETE');
    });
    document.getElementById('dice-clear').addEventListener('click', function () { hand = {}; renderHand(); });
    rollAs.addEventListener('change', renderHand);

    // Lists every character the viewer may roll as, keeping the current choice when possible.
    function updateRollers(state) {
      var previous = rollAs.value;
      rollAs.replaceChildren();
      if (state.is_owner) rollAs.add(new Option('DM', 'dm'));
      state.members.filter(function (member) { return member.can_roll; }).forEach(function (member) {
        rollAs.add(new Option(member.name + (member.kind === 'npc' ? ' (NPC)' : ''), member.character_id));
      });
      if (previous && Array.prototype.some.call(rollAs.options, function (o) { return o.value === previous; })) rollAs.value = previous;
      document.getElementById('dice-roller').hidden = rollAs.options.length === 0;
      renderHand();
    }

    rollButton.addEventListener('click', async function () {
      var groups = DICE.filter(function (sides) { return hand[sides]; });
      if (!groups.length || !rollAs.value) return;
      var total = 0;
      var details = groups.map(function (sides) {
        var results = [];
        for (var i = 0; i < hand[sides]; i++) { results.push(rollDie(sides)); }
        total += results.reduce(function (sum, value) { return sum + value; }, 0);
        return hand[sides] + 'd' + sides + ': ' + results.join(', ');
      });
      var label = 'Rolled ' + groups.map(function (sides) { return hand[sides] + 'd' + sides; }).join(' + ');
      rollButton.disabled = true;
      try {
        await api('/api/campaigns/' + encodeURIComponent(id) + '/rolls', { method: 'POST', body: { character_id: rollAs.value, label: label, detail: details.join(' | '), total: total, note: '' } });
        hand = {};
        setStatus('');
      } catch (error) { setStatus(error.message, true); }
      renderHand();
    });
    renderHand();

    document.getElementById('share-copy').addEventListener('click', async function () {
      var input = document.getElementById('share-link');
      try { await navigator.clipboard.writeText(input.value); setStatus('Join link copied.'); }
      catch (error) { input.select(); setStatus('Press copy to share the selected link.'); }
    });
    document.getElementById('campaign-leave').addEventListener('click', async function () {
      if (!window.confirm('Delete this campaign for everyone?')) return;
      try { await api('/api/campaigns/' + encodeURIComponent(id), { method: 'DELETE' }); location.href = '/campaigns'; }
      catch (error) { setStatus(error.message, true); }
    });

    resync();
  }

  if (mode === 'join') initJoin();
  else if (mode === 'view') initView();
  else initList();
})();
