// Upload functionality
const uploadArea = document.getElementById('upload-area');
const fileInput = document.getElementById('file-input');
const uploadBtn = document.getElementById('upload-btn');
const uploadForm = document.getElementById('upload-form');

// Click to select files
uploadArea.addEventListener('click', () => {
    fileInput.click();
});

// Drag and drop
uploadArea.addEventListener('dragover', (e) => {
    e.preventDefault();
    uploadArea.classList.add('dragover');
});

uploadArea.addEventListener('dragleave', () => {
    uploadArea.classList.remove('dragover');
});

uploadArea.addEventListener('drop', (e) => {
    e.preventDefault();
    uploadArea.classList.remove('dragover');
    
    const files = e.dataTransfer.files;
    fileInput.files = files;
    updateUploadButton();
});

// File input change
fileInput.addEventListener('change', updateUploadButton);

function updateUploadButton() {
    const hasFiles = fileInput.files.length > 0;
    uploadBtn.disabled = !hasFiles;
    
    if (hasFiles) {
        const count = fileInput.files.length;
        uploadBtn.textContent = `Upload ${count} Photo${count > 1 ? 's' : ''}`;
    } else {
        uploadBtn.textContent = 'Upload Photos';
    }
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

// Prevent form submission if no files selected
uploadForm.addEventListener('submit', (e) => {
    if (fileInput.files.length === 0) {
        e.preventDefault();
        alert('Please select photos to upload');
    }
});