// Buka/tutup sidebar di layar kecil
const toggle = document.getElementById('menu-toggle');
const sidebar = document.getElementById('sidebar');
if (toggle && sidebar) {
  toggle.addEventListener('click', () => sidebar.classList.toggle('open'));
}

// Konfirmasi sebelum aksi penting: tambahkan data-confirm="pesan" pada <form>
document.querySelectorAll('form[data-confirm]').forEach((form) => {
  form.addEventListener('submit', (e) => {
    if (!confirm(form.dataset.confirm)) e.preventDefault();
  });
});