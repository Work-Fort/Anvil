import { useEffect, useState } from 'react';
import { Github, BookOpen, Download } from 'lucide-react';

const REPO_OWNER = 'get-virgil';
const REPO_NAME = 'cracker-barrel';

export default function Navigation() {
  const [isScrolled, setIsScrolled] = useState(false);

  useEffect(() => {
    const handleScroll = () => {
      setIsScrolled(window.scrollY > 100);
    };
    window.addEventListener('scroll', handleScroll, { passive: true });
    return () => window.removeEventListener('scroll', handleScroll);
  }, []);

  return (
    <nav 
      className={`fixed top-0 left-0 right-0 z-[100] transition-all duration-300 ${
        isScrolled 
          ? 'bg-[#07070A]/90 backdrop-blur-md border-b border-white/5' 
          : 'bg-transparent'
      }`}
    >
      <div className="w-full px-6 lg:px-12">
        <div className="flex items-center justify-between h-16">
          {/* Logo */}
          <a
            href="#"
            onClick={(e) => {
              e.preventDefault();
              window.scrollTo({ top: 0, behavior: 'smooth' });
            }}
            className="text-[#F4F6FA] font-bold text-lg tracking-wider hover:text-[#B9FF2C] transition-colors cursor-pointer"
          >
            CRACKER BARREL
          </a>

          {/* Nav Links */}
          <div className="flex items-center gap-8">
            <a
              href="./docs/index.html"
              className="hidden sm:flex items-center gap-2 text-[#A7ACB8] hover:text-[#B9FF2C] transition-colors text-sm font-medium"
            >
              <BookOpen className="w-4 h-4" />
              Docs
            </a>
            <a
              href={`https://github.com/${REPO_OWNER}/${REPO_NAME}`}
              target="_blank"
              rel="noopener noreferrer"
              className="hidden sm:flex items-center gap-2 text-[#A7ACB8] hover:text-[#B9FF2C] transition-colors text-sm font-medium"
            >
              <Github className="w-4 h-4" />
              GitHub
            </a>
            <a
              href="#install"
              className="flex items-center gap-2 bg-[#B9FF2C] text-[#07070A] px-4 py-2 rounded-full text-sm font-semibold hover:brightness-110 transition-all hover:-translate-y-0.5"
            >
              <Download className="w-4 h-4" />
              Download
            </a>
          </div>
        </div>
      </div>
    </nav>
  );
}
