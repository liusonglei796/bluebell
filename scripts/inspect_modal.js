(() => {
  const buttons = Array.from(document.querySelectorAll('button'));
  const createTableBtn = buttons.find(b => b.innerText.trim() === 'Create table' && b.offsetParent !== null);
  if (createTableBtn) {
    createTableBtn.click();
    return { status: 'clicked_create_table' };
  }
  return { status: 'button_not_found', allButtons: buttons.map(b => b.innerText.trim()).filter(Boolean) };
})()
