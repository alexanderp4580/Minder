// Real provider, actual signed ingress and temporary SQLite. No production data.
const assert = require('node:assert/strict');
const fs = require('node:fs');

(async () => {
  const { signedHeaders } = await import('../email-worker/worker.mjs');
  const url = process.env.MINDER_URL;
  const key = Uint8Array.from(Buffer.from(process.env.MINDER_EMAIL_TEST_KEY, 'base64'));
  const get = async path => {
    const response = await fetch(url + path);
    if (response.status === 404) return { status: 'processing' };
    assert.equal(response.status, 200);
    return response.json();
  };
  const post = (path, body) => fetch(url + path, {
    method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body),
  });
  const until = async (label, read, ready) => {
    for (let attempt = 0; attempt < 300; attempt++) {
      const value = await read();
      if (ready(value)) return value;
      assert.notEqual(value.status, 'failed', `${label}: review failed`);
      await new Promise(resolve => setTimeout(resolve, 500));
    }
    throw new Error(`Timed out waiting for ${label}`);
  };
  const appointments = [
    ['Initial consultation', '2026-12-02T09:00:00+01:00', 60],
    ['Contact lens fitting', '2026-12-09T10:00:00+01:00', 45],
    ['Follow-up', '2026-12-16T11:00:00+01:00', 30],
  ];
  const body = [
    'Dear patient, here are your three appointments at Example Clinic, Main Street 1, Utrecht.',
    'Initial consultation: Wednesday 2 December 2026, 09:00, lasting one hour.',
    'Contact lens fitting: Wednesday 9 December 2026, 10:00, lasting 45 minutes.',
    'Follow-up: Wednesday 16 December 2026, 11:00, lasting 30 minutes.',
    '',
    ...Array.from({ length: 70 }, () => 'Clinic information: the reception desk is inside the main entrance. Waiting rooms and accessible facilities are on the ground floor.'),
    '',
    'Required preparation: complete and return the patient questionnaire before your first appointment. Complete it once for this course of appointments.',
    'The questionnaire deadline is 2 December 2026 at 09:00 Europe/Amsterdam, before the initial consultation begins.',
    'The questionnaire asks for current medication, allergies and contact details.',
  ].join('\n');
  assert.ok(body.length > 8000, 'Exercise a long source, not just a short appointment list');
  const raw = Buffer.from('From: Example Clinic <owner@example.com>\r\nDate: Sat, 14 Nov 2026 10:00:00 +0100\r\nSubject: Your appointments and required preparation\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n' + body);
  const headers = await signedHeaders({ key }, {
    from: 'owner@example.com', to: 'assistant@example.net', dmarc: 'pass',
  }, raw, Math.floor(Date.now() / 1000), 'three-visits-required-questionnaire');
  const ingested = await fetch(url + '/v1/email/ingest', { method: 'POST', headers, body: raw });
  assert.equal(ingested.status, 201);
  const sourceID = (await ingested.json()).capture_id;
  const ready = c => c.candidates?.length > 0 && ['ready-for-review', 'needs-answer'].includes(c.status);

  function checkFourItems(capture, label) {
    assert.equal(capture.candidates.length, 4, `${label}: three appointments and one questionnaire need four candidates`);
    for (const candidate of capture.candidates) {
      for (const boundary of ['start', 'end', 'due']) {
        assert.ok(!(candidate.proposal[boundary + '_at'] && candidate.proposal[boundary + '_date']), `${label}: ${boundary} is a date OR a timestamp`);
      }
    }
    const tasks = capture.candidates.filter(c => c.proposal.kind === 'task');
    assert.equal(tasks.length, 1, `${label}: questionnaire is a separate task`);
    const task = tasks[0];
    assert.match(task.proposal.title, /questionnaire|patient form/i, `${label}: required preparation kept`);
    for (const [title, start, duration] of appointments) {
      const visit = capture.candidates.find(c => Date.parse(c.proposal.start_at) === Date.parse(start));
      assert.ok(visit, `${label}: ${title} keeps its source start`);
      assert.equal(visit.proposal.kind, 'event');
      const actualDuration = visit.proposal.duration_minutes || (Date.parse(visit.proposal.end_at) - Date.parse(visit.proposal.start_at)) / 60000;
      assert.equal(actualDuration, duration, `${label}: ${title} keeps its stated length`);
    }
    assert.ok(task.question?.trim(), `${label}: ask when to do the questionnaire`);
    assert.equal(task.options?.length, 3, `${label}: offer three useful timing choices`);
    const choices = [
      { days: 1, match: /(?:1|one|a|the)\s+day\s+(?:before|earlier)|(?:1|01)\s+dec|dec(?:ember)?\s+(?:1|01)\b|2026-12-01/i },
      { days: 2, match: /(?:2|two)\s+days?\s+(?:before|earlier)|30\s+nov|nov(?:ember)?\s+30\b|2026-11-30/i },
      { days: 7, match: /(?:1|one|a)\s+week\s+(?:before|earlier)|7\s+days?\s+(?:before|earlier)|25\s+nov|nov(?:ember)?\s+25\b|2026-11-25/i },
    ];
    for (const choice of choices) {
      choice.option = task.options.find(option => choice.match.test(option));
      assert.ok(choice.option, `${label}: offer ${choice.days} days before the FIRST appointment; received ${JSON.stringify(task.options)}`);
    }
    return { task, choices };
  }

  let capture = await until('forwarded review', () => get('/v1/captures/' + sourceID), ready);
  const forwarded = checkFourItems(capture, 'forwarded');
  // Check the pasted copy before choosing a time. Otherwise the agent correctly
  // remembers that choice and need not ask the same question a second time.
  const dismissed = await post(`/v1/captures/${sourceID}/dismiss`, { revision: capture.revision });
  assert.equal(dismissed.status, 200);
  const pasted = await post('/v1/chat/messages', { text: 'Please propose the appointments and tasks from this email for review.\n\n' + body });
  assert.equal(pasted.status, 202);
  const pastedID = (await pasted.json()).entry.id;
  capture = await until('pasted review', () => get('/v1/captures/' + pastedID), ready);
  const pastedResult = checkFourItems(capture, 'pasted');
  // Walk review order before answering the questionnaire. These are local
  // discards; no owner items or notifications are created by this test.
  while (capture.candidates.find(c => c.id === pastedResult.task.id).state === 'queued') {
    const current = capture.candidates.find(c => c.state === 'pending');
    assert.ok(current?.change_id, 'An appointment preview precedes the questionnaire');
    const discarded = await post(`/v1/changes/${current.change_id}/discard`);
    assert.equal(discarded.status, 200);
    capture = await get('/v1/captures/' + pastedID);
  }
  const unchangedVisits = capture.candidates.filter(c => c.id !== pastedResult.task.id).map(c => [c.id, c.proposal]);
  const answered = await post(`/v1/captures/${pastedID}/answers`, {
    revision: capture.revision, candidate_id: pastedResult.task.id, text: 'Actually, make the task itself 3 days before the first appointment.',
  });
  assert.equal(answered.status, 200);
  capture = await until('questionnaire timing answer', async () => {
    const history = await get('/v1/chat?after=0&limit=300');
    assert.ok(!history.some(entry => entry.kind === 'error'), 'Questionnaire answer must commit without a chat error');
    return get('/v1/captures/' + pastedID);
  }, c => c.revision > capture.revision && ready(c));
  assert.equal(capture.candidates.length, 4, 'Answer keeps all four candidates');
  const updated = capture.candidates.find(c => c.id === pastedResult.task.id);
  assert.ok(updated, 'Answer updates the existing task');
  const proposal = updated.proposal;
  const day = value => new Date(value).toLocaleDateString('en-CA', { timeZone: 'Europe/Amsterdam' });
  assert.equal(day(proposal.start_at || proposal.start_date), '2026-11-29', 'Custom reply schedules the task itself three days before the first appointment');
  assert.equal(day(proposal.due_at || proposal.due_date), '2026-12-02', 'The source deadline stays separate from the work date');
  const anchor = Date.parse(proposal.due_at || proposal.start_at || ((proposal.due_date || proposal.start_date || '') + 'T09:00:00+01:00'));
  const reminders = proposal.reminder_offsets_minutes?.length
    ? proposal.reminder_offsets_minutes.map(minutes => anchor - minutes * 60000)
    : proposal.remind_at ? [Date.parse(proposal.remind_at)] : [];
  assert.ok(reminders.some(at => day(at) === '2026-11-29'), 'The effective reminder follows the chosen work date, including offset precedence');
  assert.ok(reminders.every(at => at < Date.parse(appointments[0][1])), 'Questionnaire reminders precede the first appointment');
  for (const [id, prior] of unchangedVisits) assert.deepEqual(capture.candidates.find(c => c.id === id)?.proposal, prior, 'Timing answer leaves other appointments unchanged');
  assert.ok(updated.change_id, 'A valid pending preview follows the custom answer');
  const confirmed = await post(`/v1/changes/${updated.change_id}/confirm`);
  assert.equal(confirmed.status, 200, 'Confirm saves the scheduled task');
  const items = await get('/v1/items?from=2026-11-01T00:00:00Z&to=2026-12-31T00:00:00Z');
  const stored = items.find(item => item.kind === 'task');
  assert.ok(stored);
  assert.equal(day(stored.start_at || stored.start_date), '2026-11-29');
  assert.equal(day(stored.due_at || stored.due_date), '2026-12-02');
  // The same intent after Confirm is an ordinary composer message, not a review answer.
  const moved = await post('/v1/chat/messages', {text:'Move work on the questionnaire to 4 days before the first appointment. Keep the deadline and notify me at the work time.'});
  assert.equal(moved.status, 202);
  let movePreview;
  for (let attempt=0; attempt<300; attempt++) {
    const history=await get('/v1/chat?after=0&limit=300');
    assert.ok(!history.some(entry=>entry.kind==='error'),'Moving a confirmed task must not fail');
    movePreview=history.find(entry=>entry.kind==='preview'&&entry.change?.action==='update_item'&&entry.change.state==='pending');
    if(movePreview)break;
    await new Promise(resolve=>setTimeout(resolve,500));
  }
  assert.ok(movePreview,'A normal chat correction proposes a real task-date update');
  assert.equal((await post(`/v1/changes/${movePreview.change.id}/confirm`)).status,200);
  const movedTask=(await get('/v1/items?from=2026-11-01T00:00:00Z&to=2026-12-31T00:00:00Z')).find(item=>item.id===stored.id);
  assert.equal(day(movedTask.start_at||movedTask.start_date),'2026-11-28','Confirmed task moves its scheduled date');
  assert.equal(day(movedTask.due_at||movedTask.due_date),'2026-12-02','Moving work keeps the source deadline');

  const result = {
    forwardedItems: 4, pastedItems: 4, statedDurationsKept: true,
    forwardedQuestion: forwarded.task.question, forwardedOptions: forwarded.task.options,
    pastedQuestion: pastedResult.task.question, pastedOptions: pastedResult.task.options,
    taskScheduledThreeDaysBefore: true, deadlineKept: true,
    confirmedTaskMovedFromMainChat: true,
    selectedReminderBeforeFirstAppointment: true, otherAppointmentsUnchanged: true,
  };
  fs.writeFileSync('.local/real-model-preparation-results.json', JSON.stringify(result, null, 2));
  console.log(JSON.stringify(result));
})().catch(error => { console.error(error); process.exitCode = 1; });
