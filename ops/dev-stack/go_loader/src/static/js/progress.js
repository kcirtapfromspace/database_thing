const newLocal = form.addEventListener('submit', e => {
    e.preventDefault();

    const formData = new FormData(form);
    const xhr = new XMLHttpRequest();
    const form = document.querySelector('form');
    const fileInput = document.querySelector('#file');
    const errorMessage = document.querySelector('.alert-danger');
    const successMessage = document.querySelector('.alert-success');
    const progressContainer = document.querySelector('.progress-container');
    const progressBar = document.querySelector('.progress-bar');
    const container = document.querySelector('.container');

    if (progressContainer) {
        progressContainer.style.display = 'block';
    }

    if (fileInput) {
        fileInput.addEventListener('dragover', e => {
            e.preventDefault();
            fileInput.style.background = '#eee';
        });

        fileInput.addEventListener('dragleave', e => {
            e.preventDefault();
            fileInput.style.background = '#fff';
        });

        fileInput.addEventListener('drop', e => {
            e.preventDefault();
            fileInput.style.background = '#fff';
            if (fileInput.files) {
                fileInput.files = e.dataTransfer.files;
            }
        });
    }

    if (form) {
        form.addEventListener('submit', e => {
            e.preventDefault();

            const formData = new FormData(form);
            const xhr = new XMLHttpRequest();

            xhr.open('POST', '/upload', true);

            xhr.upload.onprogress = function (e) {
                if (e.lengthComputable) {
                    progressContainer.style.display = 'block';
                    progressBar.style.width = `${(e.loaded / e.total) * 100}%`;
                }
            };
            xhr.onloadstart = function () {
                form.style.opacity = 0.5;
                successMessage.style.display = 'none';
                errorMessage.style.display = 'none';
                progressContainer.style.display = 'block';
            };

            xhr.onload = function () {
                if (this.status === 200) {
                    if (progressContainer) {
                        progressContainer.style.display = 'none';
                    }
                    if (progressBar) {
                        progressBar.style.width = '0%';
                    }
                    if (successMessage) {
                        successMessage.style.display = 'block';
                    }
                    if (form) {
                        form.reset();
                    }
                } else {
                    if (errorMessage) {
                        errorMessage.textContent = this.responseText;
                        errorMessage.style.display = 'block';
                    }
                }
            };

            xhr.send(formData);
        });
    }
});
