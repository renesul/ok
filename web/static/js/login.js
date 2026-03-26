document.getElementById('loginForm').addEventListener('submit', function (event) {
  event.preventDefault();

  var form = event.target;
  var password = form.password.value;
  var errorElement = document.getElementById('errorBox');

  errorElement.classList.remove('visible');

  if (!password) {
    showError('Enter your password.');
    return;
  }

  fetch('/api/auth/login', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ password: password })
  })
    .then(function (response) {
      if (!response.ok) {
        return response.json().then(function (data) {
          throw new Error(data.error || 'Login failed.');
        });
      }
      return response.json();
    })
    .then(function () {
      window.location.href = '/chat';
    })
    .catch(function (error) {
      showError(error.message);
    });

  function showError(message) {
    errorElement.textContent = message;
    errorElement.classList.add('visible');
  }
});
