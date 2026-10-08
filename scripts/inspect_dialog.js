(() => {
  const dialog = document.querySelector('[role="dialog"]');
  if (!dialog) return { hasDialog: false };
  const inputs = Array.from(dialog.querySelectorAll('input, button, textarea, select')).map(el => ({
    tag: el.tagName.toLowerCase(),
    id: el.id,
    type: el.type,
    name: el.name,
    placeholder: el.placeholder,
    text: el.innerText ? el.innerText.trim() : '',
    value: el.value,
  }));
  return {
    hasDialog: true,
    title: dialog.querySelector('h1, h2, h3')?.innerText,
    elements: inputs,
  };
})()
