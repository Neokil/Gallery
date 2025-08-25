// Upload functionality
const uploadArea = document.getElementById('upload-area');
const fileInput = document.getElementById('file-input');
const uploadForm = document.getElementById('upload-form');
const uploadContent = document.getElementById('upload-content');
const uploadDialog = document.getElementById('upload-dialog');
const fileList = document.getElementById('file-list');
const uploadSubmitBtn = document.getElementById('upload-submit-btn');

// Click to select files or drag and drop - opens dialog
uploadArea.addEventListener('click', () => {
    if (!uploadArea.classList.contains('uploading')) {
        openUploadDialog();
    }
});

// Drag and drop
uploadArea.addEventListener('dragover', (e) => {
    e.preventDefault();
    if (!uploadArea.classList.contains('uploading')) {
        uploadArea.classList.add('dragover');
    }
});

uploadArea.addEventListener('dragleave', () => {
    uploadArea.classList.remove('dragover');
});

uploadArea.addEventListener('drop', (e) => {
    e.preventDefault();
    uploadArea.classList.remove('dragover');

    if (!uploadArea.classList.contains('uploading')) {
        const files = e.dataTransfer.files;
        fileInput.files = files;
        openUploadDialog();
        updateFileList();
    }
});

// File input change
fileInput.addEventListener('change', () => {
    updateFileList();
});

// Dialog functions
function openUploadDialog() {
    uploadDialog.style.display = 'flex';
    updateFileList();
}

function closeUploadDialog() {
    uploadDialog.style.display = 'none';
    fileInput.value = '';
    document.getElementById('event-name').value = '';
    updateFileList();
}

function selectFiles() {
    fileInput.click();
}

function updateFileList() {
    const files = fileInput.files;
    uploadSubmitBtn.disabled = files.length === 0;

    if (files.length === 0) {
        fileList.innerHTML = '<p class="no-files">No photos selected</p>';
        return;
    }

    let html = '';
    for (let i = 0; i < files.length; i++) {
        const file = files[i];
        const size = formatFileSize(file.size);
        html += `
            <div class="file-item">
                <span class="file-name">${file.name}</span>
                <span class="file-size">${size}</span>
            </div>
        `;
    }
    fileList.innerHTML = html;
}

function formatFileSize(bytes) {
    if (bytes === 0) return '0 Bytes';
    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

function uploadFiles() {
    const files = fileInput.files;
    const eventName = document.getElementById('event-name').value.trim();

    if (files.length === 0) return;

    // Create FormData BEFORE closing dialog to preserve files
    const formData = new FormData();
    for (let i = 0; i < files.length; i++) {
        formData.append('photos', files[i]);
    }
    if (eventName) {
        formData.append('event_name', eventName);
    }

    // Show uploading state
    uploadArea.classList.add('uploading');
    const count = files.length;
    uploadContent.innerHTML = `
        <div class="upload-spinner"></div>
        <p>Uploading ${count} photo${count > 1 ? 's' : ''}...</p>
    `;

    // Close dialog after creating FormData
    closeUploadDialog();

    fetch('/upload', {
        method: 'POST',
        body: formData
    })
        .then(response => {
            if (response.ok) {
                // Success - reload the page to show new photos
                window.location.reload();
            } else {
                throw new Error('Upload failed');
            }
        })
        .catch(error => {
            console.error('Upload error:', error);
            // Reset upload area on error
            resetUploadArea();
            alert('Upload failed. Please try again.');
        });
}

function resetUploadArea() {
    uploadArea.classList.remove('uploading');
    uploadContent.innerHTML = `
        <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"></path>
            <polyline points="7,10 12,15 17,10"></polyline>
            <line x1="12" y1="15" x2="12" y2="3"></line>
        </svg>
        <p>Drag photos here or click to select</p>
    `;
    fileInput.value = '';
}

// Modal functionality
function openModal(imageSrc) {
    const modal = document.getElementById('modal');
    const modalImg = document.getElementById('modal-img');

    modal.style.display = 'block';
    modalImg.src = imageSrc;
}

function closeModal() {
    document.getElementById('modal').style.display = 'none';
}

// Close modal with Escape key
document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') {
        closeModal();
    }
});

// Handle form submission
uploadForm.addEventListener('submit', (e) => {
    e.preventDefault();
    uploadFiles();
});

// Close dialog with Escape key
document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') {
        closeModal();
        if (uploadDialog.style.display === 'flex') {
            closeUploadDialog();
        }
    }
});

