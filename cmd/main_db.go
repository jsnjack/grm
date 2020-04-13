package cmd

import (
	bolt "go.etcd.io/bbolt"
)

func saveToDB(pkg *Package, filter string, filename string, version string) error {
	err := DB.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(PackagesBucket)

		pb, err := bucket.CreateBucketIfNotExists([]byte(pkg.GetFullName()))
		if err != nil {
			return err
		}
		err = pb.Put([]byte("filename"), []byte(filename))
		if err != nil {
			return err
		}
		err = pb.Put([]byte("filter"), []byte(filter))
		if err != nil {
			return err
		}
		err = pb.Put([]byte("version"), []byte(version))
		if err != nil {
			return err
		}
		return nil
	})
	return err
}

func loadAllInstalledFromDB() ([]*Package, error) {
	var pkgs []*Package
	err := DB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(PackagesBucket))
		c := b.Cursor()
		for key, _ := c.First(); key != nil; key, _ = c.Next() {
			pb := b.Bucket(key)
			if pb != nil {
				p, err := createPackageFromDB(string(key), pb)
				if err != nil {
					continue
				}
				pkgs = append(pkgs, p)
			}
		}
		return nil
	})
	return pkgs, err
}

func createPackageFromDB(name string, b *bolt.Bucket) (*Package, error) {
	p, err := CreatePackage(name)
	if err != nil {
		return nil, err
	}
	p.Version = string(b.Get([]byte("version")))
	p.Filter = string(b.Get([]byte("filter")))
	locked := b.Get([]byte("locked"))
	if locked != nil {
		p.Locked = string(b.Get([]byte("lock")))
	}
	return p, nil
}
